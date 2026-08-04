import json
import sys

import torch
from safetensors.torch import save_file


weights_path = sys.argv[1]
torch.set_num_threads(1)

x = torch.tensor(
    [
        0.25, -0.5, 1.0, 0.75,
        -1.25, 0.5, 0.25, 1.5,
        0.6, -0.2, 0.9, -0.7,
        -0.4, 1.2, -0.8, 0.3,
    ],
    dtype=torch.float32,
).reshape(2, 2, 4)
labels = torch.tensor([1, 2], dtype=torch.int64)


def matrix(rows, cols, offset):
    return (torch.arange(rows * cols, dtype=torch.float32).reshape(rows, cols) + offset) / 17


weights = {
    "Wq": matrix(4, 4, 1),
    "Wk": matrix(4, 4, 3),
    "Wv": matrix(4, 4, 5),
    "Wo": matrix(4, 4, 7),
    "bq": torch.tensor([0.01, -0.02, 0.03, -0.04], dtype=torch.float32),
    "bk": torch.tensor([-0.03, 0.02, -0.01, 0.04], dtype=torch.float32),
    "bv": torch.tensor([0.02, 0.01, -0.04, 0.03], dtype=torch.float32),
    "bo": torch.tensor([0.01, 0.02, -0.02, -0.01], dtype=torch.float32),
    "W1": matrix(4, 6, 9),
    "b1": torch.tensor([0.01, -0.01, 0.02, -0.02, 0.03, -0.03], dtype=torch.float32),
    "W2": matrix(6, 4, 11),
    "b2": torch.tensor([0.03, -0.02, 0.01, -0.04], dtype=torch.float32),
    "gamma1": torch.tensor([1.0, 0.9, 1.1, 0.8], dtype=torch.float32),
    "beta1": torch.tensor([0.0, 0.1, -0.1, 0.05], dtype=torch.float32),
    "gamma2": torch.tensor([0.95, 1.05, 0.85, 1.15], dtype=torch.float32),
    "beta2": torch.tensor([0.02, -0.03, 0.04, -0.05], dtype=torch.float32),
    "head_w": matrix(4, 3, 13),
    "head_b": torch.tensor([0.02, -0.01, 0.03], dtype=torch.float32),
}
parameter_names = list(weights)
parameters = [weights[name].detach().clone().requires_grad_() for name in parameter_names]
weights = dict(zip(parameter_names, parameters))
save_file({name: value.detach() for name, value in weights.items()}, weights_path)

heads, head_size = 2, 2
q_shape = (2, 2, heads, head_size)
q = (x @ weights["Wq"] + weights["bq"]).reshape(q_shape).transpose(1, 2)
k = (x @ weights["Wk"] + weights["bk"]).reshape(q_shape).permute(0, 2, 3, 1)
scores = q @ k / (2.0**0.5)
probability = torch.softmax(scores, dim=-1)
v = (x @ weights["Wv"] + weights["bv"]).reshape(q_shape).transpose(1, 2)
context = (probability @ v).transpose(1, 2).reshape(2, 2, 4)
attention = context @ weights["Wo"] + weights["bo"]
first_residual = x + attention
first_norm = torch.nn.functional.layer_norm(
    first_residual, (4,), weights["gamma1"], weights["beta1"], 1e-5
)
hidden = torch.nn.functional.gelu(first_norm @ weights["W1"] + weights["b1"], approximate="none")
feed_forward = hidden @ weights["W2"] + weights["b2"]
second_residual = first_norm + feed_forward
encoded = torch.nn.functional.layer_norm(
    second_residual, (4,), weights["gamma2"], weights["beta2"], 1e-5
)
pooled = encoded.mean(dim=1)
logits = pooled @ weights["head_w"] + weights["head_b"]
loss = torch.nn.functional.cross_entropy(logits, labels, reduction="mean")
loss.backward()

gradients = {name: value.grad.detach().reshape(-1).tolist() for name, value in weights.items()}
optimizer = torch.optim.Adam(parameters, lr=0.003, betas=(0.9, 0.999), eps=1e-8)
optimizer.step()
updated = {name: value.detach().reshape(-1).tolist() for name, value in weights.items()}

print(json.dumps({
    "loss": float(loss.detach()),
    "parameter_names": parameter_names,
    "gradients": gradients,
    "parameters": updated,
}, separators=(",", ":")))
