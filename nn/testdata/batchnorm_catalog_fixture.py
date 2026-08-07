import json
import sys

import torch


torch.set_num_threads(1)


def values(count, offset, divisor):
    return ((torch.arange(count, dtype=torch.float32) + offset) / divisor).tolist()


model = torch.nn.Sequential(
    torch.nn.Conv2d(1, 2, 3, padding=1),
    torch.nn.BatchNorm2d(2),
    torch.nn.ReLU(),
    torch.nn.AdaptiveAvgPool2d((1, 1)),
    torch.nn.Flatten(),
    torch.nn.Linear(2, 3),
)
with torch.no_grad():
    model[0].weight.copy_(torch.tensor(values(18, -9, 20)).reshape(2, 1, 3, 3))
    model[0].bias.copy_(torch.tensor([-0.1, 0.2]))
    model[1].weight.copy_(torch.tensor([1.1, 0.8]))
    model[1].bias.copy_(torch.tensor([-0.2, 0.3]))
    model[1].running_mean.copy_(torch.tensor([0.25, -0.4]))
    model[1].running_var.copy_(torch.tensor([1.4, 0.7]))
    model[5].weight.copy_(torch.tensor(values(6, -3, 11)).reshape(3, 2))
    model[5].bias.copy_(torch.tensor([0.05, -0.1, 0.15]))

batches = [
    ((torch.arange(2 * 1 * 4 * 4, dtype=torch.float32) - 11) / 17).reshape(2, 1, 4, 4),
    ((torch.arange(2 * 1 * 4 * 4, dtype=torch.float32) + 3) / 13).reshape(2, 1, 4, 4),
    ((torch.arange(2 * 1 * 4 * 4, dtype=torch.float32) % 19 - 8) / 11).reshape(2, 1, 4, 4),
]
labels = [torch.tensor([0, 2]), torch.tensor([1, 0]), torch.tensor([2, 1])]
optimizer = torch.optim.SGD(model.parameters(), lr=0.02)
state = {name: value.detach().reshape(-1).tolist() for name, value in model.state_dict().items()}
steps = []
for batch, target in zip(batches, labels):
    optimizer.zero_grad()
    loss = torch.nn.functional.cross_entropy(model(batch), target)
    loss.backward()
    gradients = {name: value.grad.detach().reshape(-1).tolist() for name, value in model.named_parameters()}
    optimizer.step()
    parameters = {name: value.detach().reshape(-1).tolist() for name, value in model.named_parameters()}
    steps.append(
        {
            "loss": loss.item(),
            "gradients": gradients,
            "parameters": parameters,
            "running_mean": model[1].running_mean.detach().tolist(),
            "running_var": model[1].running_var.detach().tolist(),
        }
    )

print(json.dumps({"state": state, "batches": [batch.reshape(-1).tolist() for batch in batches], "labels": [target.tolist() for target in labels], "steps": steps}))
