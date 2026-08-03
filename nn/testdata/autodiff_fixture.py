import json
import sys

import torch
from safetensors.torch import save_file


weights_path = sys.argv[1]
x = torch.tensor(
    [
        [0.2, -0.7, 1.1],
        [0.4, 0.3, -0.8],
        [-0.6, 0.9, 0.5],
        [1.2, -0.1, -0.4],
    ],
    dtype=torch.float32,
)
labels = torch.tensor([2, 0, 1, 2], dtype=torch.int64)
w1 = (torch.arange(12, dtype=torch.float32).reshape(3, 4) - 5) / 7
b1 = (torch.arange(4, dtype=torch.float32) - 1.5) / 5
w2 = (torch.arange(12, dtype=torch.float32).reshape(4, 3) - 5) / 9
b2 = torch.tensor([0.15, -0.25, 0.35], dtype=torch.float32)
for parameter in (w1, b1, w2, b2):
    parameter.requires_grad_()

save_file({"w1": w1.detach(), "b1": b1.detach(), "w2": w2.detach(), "b2": b2.detach()}, weights_path)
hidden = torch.tanh(x @ w1 + b1)
logits = hidden @ w2 + b2
loss = torch.nn.functional.cross_entropy(logits, labels, reduction="mean")
loss.backward()

print(
    json.dumps(
        {
            "loss": float(loss.detach()),
            "gradients": {
                "w1": w1.grad.detach().reshape(-1).tolist(),
                "b1": b1.grad.detach().reshape(-1).tolist(),
                "w2": w2.grad.detach().reshape(-1).tolist(),
                "b2": b2.grad.detach().reshape(-1).tolist(),
            },
        },
        separators=(",", ":"),
    )
)
