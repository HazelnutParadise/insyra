import json
import math
import sys

import torch
from safetensors.torch import save_file


f16 = torch.tensor(
    [1.1, -2.5, 2**-24, -(2**-24), float("inf"), float("-inf"), float("nan")],
    dtype=torch.float16,
)
bf16 = torch.tensor(
    [1.1, -2.5, 2**-133, -(2**-133), float("inf"), float("-inf"), float("nan")],
    dtype=torch.bfloat16,
)

save_file(
    {
        "weights": torch.tensor([[1.25, -2.5, 3.75], [4.5, 0.0, -6.25]], dtype=torch.float32),
        "indices": torch.tensor([[0, 3], [8, -1]], dtype=torch.int64),
        "mask": torch.tensor([[True, False, True], [False, True, False]], dtype=torch.bool),
        "f16": f16,
        "bf16": bf16,
    },
    sys.argv[1],
    metadata={"format": "pt", "note": "ignored by dl"},
)


def reference(tensor):
    widened = tensor.float().reshape(-1)
    values = []
    for value in widened.tolist():
        if math.isnan(value):
            values.append("NaN")
        elif math.isinf(value):
            values.append("+Inf" if value > 0 else "-Inf")
        else:
            values.append(repr(value))
    bits = [int(value) & 0xFFFFFFFF for value in widened.view(torch.int32).tolist()]
    return {"shape": list(tensor.shape), "values": values, "bits": bits}


print(json.dumps({"f16": reference(f16), "bf16": reference(bf16)}))
