"""
Thin wrapper around the HF model so main.py doesn't know model internals.
Swapping SigLIP2 <-> Jina-CLIP-v2 should only touch this file — the
ingestion (worker) and query (search-ranking-service) call sites are
model-agnostic and only depend on the /embed HTTP contract in main.py.
"""
import torch
from PIL import Image
from transformers import AutoModel, AutoProcessor

# Prompt prefixes used for query/document asymmetry, where the underlying
# model supports it (e.g. Jina-CLIP-v2). For models without native prefix
# support (e.g. base SigLIP2), these are no-ops on the image path and a
# light text prefix on the text path — cheap to keep either way so the
# HTTP contract doesn't change if you swap models later.
PROMPT_PREFIXES = {
    "query_text": "Represent this search query: ",
    "catalog_text": "Represent this product: ",
}


class Encoder:
    def __init__(self, model_name: str):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        self.model = AutoModel.from_pretrained(model_name).to(self.device).eval()
        self.processor = AutoProcessor.from_pretrained(model_name)

    @torch.inference_mode()
    def encode_image(self, image: Image.Image, prompt: str) -> torch.Tensor:
        inputs = self.processor(images=image, return_tensors="pt").to(self.device)
        features = self.model.get_image_features(**inputs)
        return features[0].cpu().float().numpy()

    @torch.inference_mode()
    def encode_text(self, text: str, prompt: str) -> torch.Tensor:
        prefixed = PROMPT_PREFIXES.get(prompt, "") + text
        inputs = self.processor(text=[prefixed], return_tensors="pt", padding=True).to(self.device)
        features = self.model.get_text_features(**inputs)
        return features[0].cpu().float().numpy()
