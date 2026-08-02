import json
import sys

import numpy as np
import onnx
from onnx import TensorProto, helper, numpy_helper
import onnxruntime as ort


def make_model(nodes, inputs, outputs, initializers=()):
    graph = helper.make_graph(
        nodes,
        "dl-parity",
        inputs,
        outputs,
        initializer=list(initializers),
    )
    model = helper.make_model(
        graph,
        producer_name="insyra-dl-parity",
        opset_imports=[helper.make_opsetid("", 13)],
    )
    model.ir_version = 9
    onnx.checker.check_model(model)
    return model


def run_model(model, feed):
    session = ort.InferenceSession(
        model.SerializeToString(),
        providers=["CPUExecutionProvider"],
    )
    values = session.run(None, feed)
    result = []
    for value in values:
        array = np.asarray(value)
        result.append({
            "shape": list(array.shape),
            "data": array.astype(np.float32).reshape(-1).tolist(),
        })
    print(json.dumps(result))


def one_op(name):
    if name == "Gemm":
        x = np.array([[1, -2, 3], [4, 5, -6]], dtype=np.float32)
        weight = np.array([[1, 2], [3, 4], [5, 6]], dtype=np.float32)
        bias = np.array([0.5, -1], dtype=np.float32)
        model = make_model(
            [helper.make_node("Gemm", ["X", "W", "B"], ["Y"], alpha=1.25, beta=0.5)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 2])],
            [numpy_helper.from_array(weight, "W"), numpy_helper.from_array(bias, "B")],
        )
        return model, {"X": x}
    if name == "MatMul":
        x = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        weight = np.array([[1, 2], [3, 4], [5, 6]], dtype=np.float32)
        model = make_model(
            [helper.make_node("MatMul", ["X", "W"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 2])],
            [numpy_helper.from_array(weight, "W")],
        )
        return model, {"X": x}
    if name in ("Add", "Sub", "Mul"):
        left = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        right = np.array([10, 20, 30], dtype=np.float32)
        model = make_model(
            [helper.make_node(name, ["A", "B"], ["Y"])],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 3]), helper.make_tensor_value_info("B", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"A": left, "B": right}
    if name in ("Relu", "Sigmoid", "Tanh", "Identity", "Softmax"):
        value = np.array([[-1, 0, 1], [2, -2, 0.5]], dtype=np.float32)
        attributes = {"axis": 1} if name == "Softmax" else {}
        model = make_model(
            [helper.make_node(name, ["X"], ["Y"], **attributes)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"X": value}
    if name == "Reshape":
        value = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        shape = np.array([3, 2], dtype=np.int64)
        model = make_model(
            [helper.make_node("Reshape", ["X", "shape"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3]), helper.make_tensor_value_info("shape", TensorProto.INT64, [2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 2])],
        )
        return model, {"X": value, "shape": shape}
    if name == "Flatten":
        value = np.arange(12, dtype=np.float32).reshape(2, 3, 2)
        model = make_model(
            [helper.make_node("Flatten", ["X"], ["Y"], axis=1)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 6])],
        )
        return model, {"X": value}
    if name == "Transpose":
        value = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Transpose", ["X"], ["Y"], perm=[1, 0])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 2])],
        )
        return model, {"X": value}
    if name == "Cast":
        value = np.array([[-1, 0, 1]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Cast", ["X"], ["Y"], to=TensorProto.FLOAT)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 3])],
        )
        return model, {"X": value}
    if name == "Constant":
        value = np.array([[1.5, -2], [3, 4.25]], dtype=np.float32)
        tensor = numpy_helper.from_array(value, "constant-value")
        model = make_model(
            [helper.make_node("Constant", [], ["Y"], value=tensor)],
            [],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 2])],
        )
        return model, {}
    raise ValueError("unknown parity operator: " + name)


def mlp_model():
    x = np.array(
        [[0.5, -1, 2], [1.5, 0.25, -0.75], [-2, 1, 0.5]],
        dtype=np.float32,
    )
    w1 = np.array([[1, -2, 0.5, 1.5], [0.25, 1, -1, 2], [-1, 0.5, 2, -0.25]], dtype=np.float32)
    b1 = np.array([0.5, -0.5, 0.25, 1], dtype=np.float32)
    w2 = np.array([[1, -1], [0.5, 2], [-2, 0.25], [1.5, -0.5]], dtype=np.float32)
    b2 = np.array([0.25, -0.75], dtype=np.float32)
    nodes = [
        helper.make_node("Gemm", ["X", "W1", "B1"], ["H"]),
        helper.make_node("Relu", ["H"], ["R"]),
        helper.make_node("Gemm", ["R", "W2", "B2"], ["Y"]),
        helper.make_node("Sigmoid", ["Y"], ["Z"]),
    ]
    model = make_model(
        nodes,
        [helper.make_tensor_value_info("X", TensorProto.FLOAT, [None, 3])],
        [helper.make_tensor_value_info("Z", TensorProto.FLOAT, [None, 2])],
        [
            numpy_helper.from_array(w1, "W1"),
            numpy_helper.from_array(b1, "B1"),
            numpy_helper.from_array(w2, "W2"),
            numpy_helper.from_array(b2, "B2"),
        ],
    )
    return model, {"X": x}


def main():
    mode = sys.argv[1]
    if mode == "one-op":
        model, feed = one_op(sys.argv[2])
        run_model(model, feed)
        return
    if mode == "mlp":
        model, feed = mlp_model()
        with open(sys.argv[2], "wb") as handle:
            handle.write(model.SerializeToString())
        run_model(model, feed)
        return
    raise ValueError("unknown parity mode: " + mode)


if __name__ == "__main__":
    main()
