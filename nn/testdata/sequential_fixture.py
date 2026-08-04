import json
import sys

import torch
from safetensors.torch import save_file


weights_path = sys.argv[1]
torch.set_num_threads(1)

batch = 4
input_size = 784
hidden_size = 128
classes = 10
x = (torch.arange(batch * input_size, dtype=torch.float32).reshape(batch, input_size) % 23 - 11) / 23
labels = torch.tensor([2, 0, 1, 2], dtype=torch.int64)

initial = {
    "0.weight": (torch.arange(hidden_size * input_size, dtype=torch.float32).reshape(hidden_size, input_size) % 29 - 14) / 29,
    "0.bias": (torch.arange(hidden_size, dtype=torch.float32) % 17 - 8) / 17,
    "2.weight": (torch.arange(classes * hidden_size, dtype=torch.float32).reshape(classes, hidden_size) % 31 - 15) / 31,
    "2.bias": (torch.arange(classes, dtype=torch.float32) % 7 - 3) / 7,
}
save_file({name: value.contiguous() for name, value in initial.items()}, weights_path)

model = torch.nn.Sequential(torch.nn.Linear(input_size, hidden_size), torch.nn.ReLU(), torch.nn.Linear(hidden_size, classes))
with torch.no_grad():
    model[0].weight.copy_(initial["0.weight"])
    model[0].bias.copy_(initial["0.bias"])
    model[2].weight.copy_(initial["2.weight"])
    model[2].bias.copy_(initial["2.bias"])

forward = model(x)
optimizer = torch.optim.AdamW(model.parameters(), lr=1e-3, weight_decay=1e-2)
optimizer.zero_grad(set_to_none=True)
loss = torch.nn.functional.cross_entropy(forward, labels, reduction="mean")
loss.backward()
optimizer.step()

post_parameters = {
    name: value.detach().reshape(-1).tolist()
    for name, value in model.state_dict().items()
}
print(
    json.dumps(
        {
            "input": x.tolist(),
            "labels": labels.tolist(),
            "forward": forward.detach().tolist(),
            "loss": float(loss.detach()),
            "post_parameters": post_parameters,
        },
        separators=(",", ":"),
    )
)
