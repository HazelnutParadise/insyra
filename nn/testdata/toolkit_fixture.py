import json
import sys

import torch
from safetensors.torch import save_file


weights_path = sys.argv[1]
torch.set_num_threads(1)

x = torch.tensor(
    [
        [0.2, -0.7, 1.1],
        [0.4, 0.3, -0.8],
        [-0.6, 0.9, 0.5],
        [1.2, -0.1, -0.4],
        [-0.3, 0.8, 0.6],
    ],
    dtype=torch.float32,
)
targets = torch.tensor(
    [
        [1.0, 0.0],
        [0.0, 1.0],
        [1.0, 1.0],
        [0.0, 0.0],
        [1.0, 0.0],
    ],
    dtype=torch.float32,
)
initial = {
    "w1": (torch.arange(12, dtype=torch.float32).reshape(3, 4) - 5) / 7,
    "b1": (torch.arange(4, dtype=torch.float32) - 1.5) / 5,
    "w2": (torch.arange(8, dtype=torch.float32).reshape(4, 2) - 3) / 8,
    "b2": torch.tensor([0.15, -0.25], dtype=torch.float32),
}
save_file(initial, weights_path)

parameters = [initial[name].detach().clone().requires_grad_() for name in initial]
optimizer = torch.optim.SGD(parameters, lr=1e-2, momentum=0.9)
scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(optimizer, T_max=6, eta_min=0)
steps = []
for _ in range(6):
    optimizer.zero_grad(set_to_none=True)
    hidden = torch.tanh(x @ parameters[0] + parameters[1])
    logits = hidden @ parameters[2] + parameters[3]
    loss = torch.nn.functional.binary_cross_entropy_with_logits(
        logits, targets, reduction="mean"
    )
    loss.backward()
    grad_norm = torch.nn.utils.clip_grad_norm_(parameters, 1.0)
    lr_used = optimizer.param_groups[0]["lr"]
    optimizer.step()
    steps.append(
        {
            "lr": float(lr_used),
            "loss": float(loss.detach()),
            "grad_norm": float(grad_norm),
            "parameters": {
                name: value.detach().reshape(-1).tolist()
                for name, value in zip(initial, parameters)
            },
        }
    )
    scheduler.step()

print(json.dumps({"steps": steps}, separators=(",", ":")))
