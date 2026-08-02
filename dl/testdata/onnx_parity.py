import json
import sys

import numpy as np
import onnx
from onnx import TensorProto, helper, numpy_helper
import onnxruntime as ort


def make_model(nodes, inputs, outputs, initializers=(), domains=()):
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
        opset_imports=[helper.make_opsetid("", 13)] + [helper.make_opsetid(domain, version) for domain, version in domains],
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
            "dtype": str(array.dtype),
            "data": array.reshape(-1).tolist(),
        })
    print(json.dumps(result))


def write_feed(model, feed, path):
    payload = []
    for value_info in model.graph.input:
        if value_info.name not in feed:
            continue
        array = np.asarray(feed[value_info.name])
        if array.dtype == object:
            dtype = "string"
        elif array.dtype == np.bool_:
            dtype = "bool"
        elif array.dtype == np.int64:
            dtype = "int64"
        else:
            dtype = "float32"
        payload.append({
            "name": value_info.name,
            "shape": list(array.shape),
            "dtype": dtype,
            "data": array.reshape(-1).tolist(),
        })
    with open(path, "w") as handle:
        json.dump(payload, handle)


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
    if name in ("Add", "Sub", "Mul", "Div"):
        left = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        right = np.array([2, 4, 5] if name == "Div" else [10, 20, 30], dtype=np.float32)
        model = make_model(
            [helper.make_node(name, ["A", "B"], ["Y"])],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 3]), helper.make_tensor_value_info("B", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"A": left, "B": right}
    if name == "Concat":
        left = np.array([[1], [2]], dtype=np.float32)
        right = np.array([[3, 4], [5, 6]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Concat", ["A", "B"], ["Y"], axis=1)],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 1]), helper.make_tensor_value_info("B", TensorProto.FLOAT, [2, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"A": left, "B": right}
    if name == "Unsqueeze":
        value = np.array([1, 2, 3], dtype=np.float32)
        axes = np.array([1], dtype=np.int64)
        model = make_model(
            [helper.make_node("Unsqueeze", ["X", "axes"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 1])],
            [numpy_helper.from_array(axes, "axes")],
        )
        return model, {"X": value}
    if name == "Gather":
        value = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        indices = np.array([2], dtype=np.int64)
        model = make_model(
            [helper.make_node("Gather", ["X", "indices"], ["Y"], axis=1)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 1])],
            [numpy_helper.from_array(indices, "indices")],
        )
        return model, {"X": value}
    if name == "GreaterOrEqual":
        left = np.array([1, 2, 3], dtype=np.int64)
        right = np.array(2, dtype=np.int64)
        model = make_model(
            [helper.make_node("GreaterOrEqual", ["A", "B"], ["Y"])],
            [helper.make_tensor_value_info("A", TensorProto.INT64, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.BOOL, [3])],
            [numpy_helper.from_array(right, "B")],
        )
        return model, {"A": left}
    if name == "Where":
        condition = np.array([True, False, True], dtype=np.bool_)
        left = np.array([1, 2, 3], dtype=np.float32)
        right = np.array(-1, dtype=np.float32)
        model = make_model(
            [helper.make_node("Where", ["C", "A", "B"], ["Y"])],
            [helper.make_tensor_value_info("C", TensorProto.BOOL, [3]), helper.make_tensor_value_info("A", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3])],
            [numpy_helper.from_array(right, "B")],
        )
        return model, {"C": condition, "A": left}
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
        value = np.array([[-1, 0, 1]], dtype=np.int64)
        model = make_model(
            [helper.make_node("Cast", ["X"], ["Y"], to=TensorProto.FLOAT)],
            [helper.make_tensor_value_info("X", TensorProto.INT64, [1, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 3])],
        )
        return model, {"X": value}
    if name == "OneHotEncoder":
        value = np.array(["red", "blue", "unknown"], dtype=object)
        model = make_model(
            [helper.make_node("OneHotEncoder", ["X"], ["Y"], domain="ai.onnx.ml", cats_strings=["red", "blue"], zeros=1)],
            [helper.make_tensor_value_info("X", TensorProto.STRING, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 2])],
            domains=[("ai.onnx.ml", 3)],
        )
        return model, {"X": value}
    if name == "LabelEncoder":
        value = np.array(["red", "blue", "unknown"], dtype=object)
        model = make_model(
            [helper.make_node("LabelEncoder", ["X"], ["Y"], domain="ai.onnx.ml", keys_strings=["red", "blue"], values_int64s=[1, 2], default_int64=-1)],
            [helper.make_tensor_value_info("X", TensorProto.STRING, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.INT64, [3])],
            domains=[("ai.onnx.ml", 3)],
        )
        return model, {"X": value}
    if name == "Scaler":
        value = np.array([[1, 2], [3, 4]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Scaler", ["X"], ["Y"], domain="ai.onnx.ml", offset=[-1.0, 1.0], scale=[2.0, 3.0])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 2])],
            domains=[("ai.onnx.ml", 3)],
        )
        return model, {"X": value}
    if name == "LinearRegressor":
        value = np.array([[1, 2], [3, 4]], dtype=np.float32)
        model = make_model(
            [helper.make_node("LinearRegressor", ["X"], ["Y"], domain="ai.onnx.ml", coefficients=[2.0, -1.0], intercepts=[0.5], targets=1, post_transform="NONE")],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 1])],
            domains=[("ai.onnx.ml", 3)],
        )
        return model, {"X": value}
    if name == "LinearClassifier":
        value = np.array([[1, 2], [3, 4]], dtype=np.float32)
        model = make_model(
            [helper.make_node("LinearClassifier", ["X"], ["label", "probabilities"], domain="ai.onnx.ml", classlabels_ints=[0, 1], coefficients=[2.0, -1.0, -1.0, 2.0], intercepts=[0.5, 0.1], post_transform="LOGISTIC")],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 2])],
            [helper.make_tensor_value_info("probabilities", TensorProto.FLOAT, [2, 2])],
            domains=[("ai.onnx.ml", 3)],
        )
        return model, {"X": value}
    if name in ("TreeEnsembleRegressor", "TreeEnsembleClassifier"):
        value = np.array([[-1], [1]], dtype=np.float32)
        common = dict(
            nodes_treeids=[0, 0, 0], nodes_nodeids=[0, 1, 2], nodes_featureids=[0, 0, 0],
            nodes_values=[0.0, 0.0, 0.0], nodes_modes=["BRANCH_LEQ", "LEAF", "LEAF"],
            nodes_truenodeids=[1, 0, 0], nodes_falsenodeids=[2, 0, 0],
            nodes_missing_value_tracks_true=[0, 0, 0], post_transform="NONE",
        )
        if name == "TreeEnsembleRegressor":
            common.update(n_targets=1, target_treeids=[0, 0], target_nodeids=[1, 2], target_ids=[0, 0], target_weights=[-1.0, 2.0])
            output = helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 1])
        else:
            common.update(classlabels_int64s=[0, 1], class_treeids=[0, 0, 0, 0], class_nodeids=[1, 1, 2, 2], class_ids=[0, 1, 0, 1], class_weights=[1.0, 0.0, 0.0, 1.0])
            output = helper.make_tensor_value_info("probabilities", TensorProto.FLOAT, [2, 2])
        model = make_model(
            [helper.make_node(name, ["X"], (["label", "probabilities"] if name == "TreeEnsembleClassifier" else ["Y"]), domain="ai.onnx.ml", **common)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 1])],
            [output],
            domains=[("ai.onnx.ml", 3)],
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


def roundtrip(model_path, payload_path):
    session = ort.InferenceSession(model_path, providers=["CPUExecutionProvider"])
    payload = json.load(open(payload_path))
    feed = {}
    for index, value in enumerate(payload):
        spec = session.get_inputs()[index]
        if spec.type == "tensor(string)":
            dtype = np.str_
        elif spec.type == "tensor(int64)":
            dtype = np.int64
        elif spec.type == "tensor(bool)":
            dtype = np.bool_
        else:
            dtype = np.float32
        feed[spec.name] = np.asarray(value, dtype=dtype)
    result = []
    for spec, value in zip(session.get_outputs(), session.run(None, feed)):
        array = np.asarray(value)
        result.append({
            "name": spec.name,
            "shape": list(array.shape),
            "dtype": str(array.dtype),
            "data": array.reshape(-1).tolist(),
        })
    print(json.dumps(result))


def main():
    mode = sys.argv[1]
    if mode == "one-op":
        model, feed = one_op(sys.argv[2])
        if len(sys.argv) > 3:
            with open(sys.argv[3], "wb") as handle:
                handle.write(model.SerializeToString())
        if len(sys.argv) > 4:
            write_feed(model, feed, sys.argv[4])
        run_model(model, feed)
        return
    if mode == "mlp":
        model, feed = mlp_model()
        with open(sys.argv[2], "wb") as handle:
            handle.write(model.SerializeToString())
        run_model(model, feed)
        return
    if mode == "roundtrip":
        roundtrip(sys.argv[2], sys.argv[3])
        return
    raise ValueError("unknown parity mode: " + mode)


if __name__ == "__main__":
    main()
