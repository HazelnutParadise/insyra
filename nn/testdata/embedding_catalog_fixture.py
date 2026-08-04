import json

import torch


torch.set_num_threads(1)
table = (torch.arange(12, dtype=torch.float32).reshape(4, 3) - 5) / 7
indices = torch.tensor([1, 3, 1, 0, 3], dtype=torch.int64)
upstream = (torch.arange(15, dtype=torch.float32).reshape(5, 3) - 4) / 9
table = torch.nn.Parameter(table)
output = table[indices]
loss = (output * upstream).mean()
loss.backward()
print(json.dumps({"table": table.detach().reshape(-1).tolist(), "indices": indices.tolist(), "upstream": upstream.reshape(-1).tolist(), "output": output.detach().reshape(-1).tolist(), "loss": loss.item(), "gradient": table.grad.reshape(-1).tolist()}))
