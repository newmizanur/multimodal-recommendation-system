"""
Embedding model server — FastAPI + async batching, the "easiest to start"
option in the article's serving tiers (vs TorchServe / Triton).

Loads a single unified multimodal encoder (SigLIP2 so400m by default;
swap MODEL_NAME for Jina-CLIP-v2) and serves both image and text
embedding requests from one process.

Query/document asymmetry: the `prompt` field distinguishes
"catalog_image" / "catalog_text" (used at ingestion time) from
"query_image" / "query_text" (used at query time). Models that support
distinct prefixes for each should use them here, not leave it to callers.
"""
import base64
import io
import os

from fastapi import FastAPI
from PIL import Image
from pydantic import BaseModel

from app.model import Encoder

MODEL_NAME = os.getenv("EMBEDDING_MODEL", "google/siglip2-so400m-patch14-384")

app = FastAPI(title="multimodal-recsys embedding server")
encoder = Encoder(MODEL_NAME)


class EmbedRequest(BaseModel):
    kind: str  # "image" | "text"
    prompt: str  # "catalog_image" | "query_image" | "catalog_text" | "query_text"
    image_base64: str | None = None
    text: str | None = None


class EmbedResponse(BaseModel):
    vector: list[float]
    model: str


@app.get("/healthz")
def healthz():
    return {"status": "ok", "model": MODEL_NAME}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    if req.kind == "image":
        if not req.image_base64:
            raise ValueError("image_base64 required for kind=image")
        image = Image.open(io.BytesIO(base64.b64decode(req.image_base64))).convert("RGB")
        vector = encoder.encode_image(image, prompt=req.prompt)
    elif req.kind == "text":
        if not req.text:
            raise ValueError("text required for kind=text")
        vector = encoder.encode_text(req.text, prompt=req.prompt)
    else:
        raise ValueError(f"unknown kind: {req.kind}")

    return EmbedResponse(vector=vector.tolist(), model=MODEL_NAME)
