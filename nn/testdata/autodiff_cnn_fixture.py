import json
import sys

import torch
from safetensors.torch import save_file


weights_path = sys.argv[1]
torch.set_num_threads(1)


def arange_tensor(shape, offset, divisor):
    count = 1
    for dimension in shape:
        count *= dimension
    return (torch.arange(count, dtype=torch.float32).reshape(shape) + offset) / divisor


def vector(length, offset, divisor):
    return arange_tensor((length,), offset, divisor)


weights = {
    "conv1.weight": arange_tensor((8, 1, 3, 3), 1, 1000),
    "conv1.bias": vector(8, 0, 100),
    "bn.weight": vector(8, 8, 10),
    "bn.bias": vector(8, -4, 100),
    "bn.running_mean": vector(8, -3, 20),
    "bn.running_var": vector(8, 10, 10),
    "conv2.weight": arange_tensor((16, 8, 3, 3), 1, 10000),
    "conv2.bias": vector(16, -8, 100),
    "fc.weight": arange_tensor((10, 16), 1, 500),
    "fc.bias": vector(10, -5, 100),
}
parameter_names = [
    "conv1.weight",
    "conv1.bias",
    "bn.weight",
    "bn.bias",
    "conv2.weight",
    "conv2.bias",
    "fc.weight",
    "fc.bias",
]
parameters = [weights[name].detach().clone().requires_grad_() for name in parameter_names]
weights.update(dict(zip(parameter_names, parameters)))
save_file(weights, weights_path)

x = ((torch.arange(4 * 8 * 8, dtype=torch.float32) % 23) - 11).reshape(4, 1, 8, 8) / 23
labels = torch.tensor([0, 3, 6, 9], dtype=torch.int64)

hidden = torch.nn.functional.conv2d(x, weights["conv1.weight"], weights["conv1.bias"], padding=1)
hidden = torch.nn.functional.batch_norm(
    hidden,
    weights["bn.running_mean"],
    weights["bn.running_var"],
    weights["bn.weight"],
    weights["bn.bias"],
    training=False,
    momentum=0.1,
    eps=1e-5,
)
hidden = torch.relu(hidden)
hidden = torch.nn.functional.max_pool2d(hidden, kernel_size=2, stride=2)
hidden = torch.nn.functional.conv2d(input=hidden, weight=weights["conv2.weight"], bias=weights["conv2.bias"], padding=1)
hidden = torch.relu(hidden)
hidden = torch.nn.functional.adaptive_avg_pool2d(hidden, (1, 1))
logits = hidden.flatten(1) @ weights["fc.weight"].transpose(0, 1) + weights["fc.bias"]
loss = torch.nn.functional.cross_entropy(logits, labels, reduction="mean")
loss.backward()

gradients = {name: weights[name].grad.detach().reshape(-1).tolist() for name in parameter_names}
optimizer = torch.optim.Adam(parameters, lr=0.003, betas=(0.9, 0.999), eps=1e-8)
optimizer.step()
updated = {name: weights[name].detach().reshape(-1).tolist() for name in parameter_names}

print(json.dumps({
    "loss": float(loss.detach()),
    "parameter_names": parameter_names,
    "gradients": gradients,
    "parameters": updated,
}, separators=(",", ":")))
