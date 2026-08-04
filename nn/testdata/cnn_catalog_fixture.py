import json
import sys

import torch
from safetensors.torch import save_file


torch.set_num_threads(1)
model = torch.nn.Sequential(
    torch.nn.Conv2d(1, 2, 3, padding=1),
    torch.nn.BatchNorm2d(2),
    torch.nn.ReLU(),
    torch.nn.MaxPool2d(2),
    torch.nn.Flatten(),
    torch.nn.Linear(2 * 2 * 2, 3),
)
with torch.no_grad():
    model[0].weight.copy_((torch.arange(18, dtype=torch.float32).reshape(2, 1, 3, 3) - 8) / 13)
    model[0].bias.copy_(torch.tensor([-0.1, 0.2]))
    model[1].weight.copy_(torch.tensor([1.2, 0.7]))
    model[1].bias.copy_(torch.tensor([-0.2, 0.4]))
    model[1].running_mean.copy_(torch.tensor([0.3, -0.5]))
    model[1].running_var.copy_(torch.tensor([0.8, 1.3]))
    model[5].weight.copy_((torch.arange(24, dtype=torch.float32).reshape(3, 8) - 11) / 17)
    model[5].bias.copy_(torch.tensor([0.05, -0.1, 0.15]))
model.eval()
input_value = (torch.arange(2 * 1 * 4 * 4, dtype=torch.float32).reshape(2, 1, 4, 4) - 9) / 19
with torch.no_grad():
    output = model(input_value)
save_file(model.state_dict(), sys.argv[1])
print(json.dumps({"input": input_value.reshape(-1).tolist(), "output": output.reshape(-1).tolist()}))
