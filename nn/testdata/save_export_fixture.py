import json
import sys

import numpy as np


def weights(path):
    import safetensors.torch
    import torch

    state = safetensors.torch.load_file(path, device="cpu")
    model = torch.nn.Sequential(
        torch.nn.Linear(3, 4),
        torch.nn.ReLU(),
        torch.nn.Linear(4, 2),
    )
    model.load_state_dict(state)
    values = model(torch.tensor([
        [0.25, -1.0, 2.0],
        [1.5, 0.0, -0.5],
    ], dtype=torch.float32)).detach().cpu().numpy()
    print(json.dumps({"shape": list(values.shape), "values": values.reshape(-1).tolist()}))


def onnx(path, input_path, shape):
    import onnxruntime as ort

    session = ort.InferenceSession(path, providers=["CPUExecutionProvider"])
    values = np.fromfile(input_path, dtype=np.float32).reshape(shape)
    output = session.run(["output"], {session.get_inputs()[0].name: values})[0]
    print(json.dumps({"shape": list(output.shape), "values": output.reshape(-1).tolist()}))


if sys.argv[1] == "weights":
    weights(sys.argv[2])
elif sys.argv[1] == "onnx":
    onnx(sys.argv[2], sys.argv[3], tuple(int(value) for value in sys.argv[4].split(",")))
else:
    raise SystemExit("unknown mode")
