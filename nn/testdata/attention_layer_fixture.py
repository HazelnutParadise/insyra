import json
import sys

import torch
from safetensors.torch import save_file


mode = sys.argv[1]
weights_path = sys.argv[2]
torch.set_num_threads(1)


def matrix(rows, cols, offset, divisor):
    return (torch.arange(rows * cols, dtype=torch.float32).reshape(rows, cols) + offset) / divisor


def fixed_input():
    return (torch.arange(2 * 5 * 16, dtype=torch.float32).reshape(2, 5, 16) - 37) / 19


if mode == "mha":
    module = torch.nn.MultiheadAttention(16, 4, batch_first=True, bias=True)
    with torch.no_grad():
        module.in_proj_weight.copy_(matrix(48, 16, -31, 23))
        module.in_proj_bias.copy_((torch.arange(48, dtype=torch.float32) - 19) / 29)
        module.out_proj.weight.copy_(matrix(16, 16, 7, 31))
        module.out_proj.bias.copy_((torch.arange(16, dtype=torch.float32) - 5) / 37)
    x = fixed_input()
    output = module(x, x, x, need_weights=False)[0]
    save_file({name: value.detach() for name, value in module.state_dict().items()}, weights_path)
    print(json.dumps({
        "input_shape": list(x.shape),
        "input": x.reshape(-1).tolist(),
        "output_shape": list(output.shape),
        "output": output.detach().reshape(-1).tolist(),
    }, separators=(",", ":")))
    raise SystemExit(0)


if mode != "encoder":
    raise ValueError(f"unknown mode {mode}")


class Residual(torch.nn.Module):
    def __init__(self, *layers):
        super().__init__()
        self.layers = torch.nn.Sequential(*layers)

    def forward(self, value):
        return value + self.layers(value)


attention = torch.nn.MultiheadAttention(16, 4, batch_first=True, bias=True)
norm1 = torch.nn.LayerNorm(16)
ffn1 = torch.nn.Linear(16, 32)
ffn2 = torch.nn.Linear(32, 16)
norm2 = torch.nn.LayerNorm(16)
head = torch.nn.Linear(16, 3)
with torch.no_grad():
    attention.in_proj_weight.copy_(matrix(48, 16, -41, 29))
    attention.in_proj_bias.copy_((torch.arange(48, dtype=torch.float32) - 17) / 47)
    attention.out_proj.weight.copy_(matrix(16, 16, 13, 37))
    attention.out_proj.bias.copy_((torch.arange(16, dtype=torch.float32) - 9) / 53)
    norm1.weight.copy_((torch.arange(16, dtype=torch.float32) + 19) / 23)
    norm1.bias.copy_((torch.arange(16, dtype=torch.float32) - 7) / 31)
    ffn1.weight.copy_(matrix(32, 16, -23, 43))
    ffn1.bias.copy_((torch.arange(32, dtype=torch.float32) - 11) / 59)
    ffn2.weight.copy_(matrix(16, 32, 5, 41))
    ffn2.bias.copy_((torch.arange(16, dtype=torch.float32) - 3) / 61)
    norm2.weight.copy_((torch.arange(16, dtype=torch.float32) + 7) / 19)
    norm2.bias.copy_((torch.arange(16, dtype=torch.float32) - 13) / 67)
    head.weight.copy_(matrix(3, 16, -5, 17))
    head.bias.copy_((torch.arange(3, dtype=torch.float32) - 1) / 13)

x = fixed_input()
labels = torch.tensor([1, 2], dtype=torch.int64)
first = x + attention(x, x, x, need_weights=False)[0]
first = torch.nn.functional.layer_norm(first, (16,), norm1.weight, norm1.bias, 1e-5)
second = first + ffn2(torch.nn.functional.gelu(ffn1(first), approximate="none"))
encoded = torch.nn.functional.layer_norm(second, (16,), norm2.weight, norm2.bias, 1e-5)
logits = head(encoded.mean(dim=1))
loss = torch.nn.functional.cross_entropy(logits, labels, reduction="mean")

parameters = [
    attention.in_proj_weight,
    attention.in_proj_bias,
    attention.out_proj.weight,
    attention.out_proj.bias,
    norm1.weight,
    norm1.bias,
    ffn1.weight,
    ffn1.bias,
    ffn2.weight,
    ffn2.bias,
    norm2.weight,
    norm2.bias,
    head.weight,
    head.bias,
]
names = [
    "0.0.in_proj_weight", "0.0.in_proj_bias", "0.0.out_proj.weight", "0.0.out_proj.bias",
    "1.weight", "1.bias",
    "2.0.weight", "2.0.bias", "2.2.weight", "2.2.bias",
    "3.weight", "3.bias", "4.weight", "4.bias",
]
state = {
    "0.0.in_proj_weight": attention.in_proj_weight.detach(),
    "0.0.in_proj_bias": attention.in_proj_bias.detach(),
    "0.0.out_proj.weight": attention.out_proj.weight.detach(),
    "0.0.out_proj.bias": attention.out_proj.bias.detach(),
    "1.weight": norm1.weight.detach(),
    "1.bias": norm1.bias.detach(),
    "2.0.weight": ffn1.weight.detach(),
    "2.0.bias": ffn1.bias.detach(),
    "2.2.weight": ffn2.weight.detach(),
    "2.2.bias": ffn2.bias.detach(),
    "3.weight": norm2.weight.detach(),
    "3.bias": norm2.bias.detach(),
    "4.weight": head.weight.detach(),
    "4.bias": head.bias.detach(),
}
save_file(state, weights_path)
loss.backward()
optimizer = torch.optim.AdamW(parameters, lr=1e-3, weight_decay=1e-2)
optimizer.step()

print(json.dumps({
    "loss": float(loss.detach()),
    "parameter_names": names,
    "gradients": {name: parameter.grad.detach().reshape(-1).tolist() for name, parameter in zip(names, parameters)},
    "parameters": {name: parameter.detach().reshape(-1).tolist() for name, parameter in zip(names, parameters)},
}, separators=(",", ":")))
