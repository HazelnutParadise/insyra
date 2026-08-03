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
    ],
    dtype=torch.float32,
)
labels = torch.tensor([2, 0, 1, 2], dtype=torch.int64)
initial = {
    "w1": (torch.arange(12, dtype=torch.float32).reshape(3, 4) - 5) / 7,
    "b1": (torch.arange(4, dtype=torch.float32) - 1.5) / 5,
    "w2": (torch.arange(12, dtype=torch.float32).reshape(4, 3) - 5) / 9,
    "b2": torch.tensor([0.15, -0.25, 0.35], dtype=torch.float32),
}
save_file(initial, weights_path)

parameters = [initial[name].detach().clone().requires_grad_() for name in initial]
optimizer = torch.optim.AdamW(parameters, lr=1e-3, weight_decay=1e-2)
scheduler = torch.optim.lr_scheduler.StepLR(optimizer, step_size=2, gamma=0.5)
steps = []
for _ in range(5):
    optimizer.zero_grad(set_to_none=True)
    hidden = torch.tanh(x @ parameters[0] + parameters[1])
    logits = hidden @ parameters[2] + parameters[3]
    loss = torch.nn.functional.cross_entropy(logits, labels, reduction="mean")
    loss.backward()
    lr_used = optimizer.param_groups[0]["lr"]
    optimizer.step()
    steps.append(
        {
            "lr": float(lr_used),
            "loss": float(loss.detach()),
            "parameters": {
                name: value.detach().reshape(-1).tolist()
                for name, value in zip(initial, parameters)
            },
        }
    )
    scheduler.step()

print(json.dumps({"steps": steps}, separators=(",", ":")))
