import json
import sys

import numpy as np
import onnx
from onnx import TensorProto, helper, numpy_helper
import onnxruntime as ort


def make_model(nodes, inputs, outputs, initializers=(), domains=(), opset=13):
    graph = helper.make_graph(
        nodes,
        "nn-parity",
        inputs,
        outputs,
        initializer=list(initializers),
    )
    model = helper.make_model(
        graph,
        producer_name="insyra-nn-parity",
        opset_imports=[helper.make_opsetid("", opset)] + [helper.make_opsetid(domain, version) for domain, version in domains],
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
        x = np.arange(1, 13, dtype=np.float32).reshape(2, 1, 2, 3)
        weight = np.array([
            1, 2, 3, 4, 5, 6,
            2, 1, 4, 3, 6, 5,
            3, 2, 5, 4, 7, 6,
        ], dtype=np.float32).reshape(1, 3, 3, 2)
        model = make_model(
            [helper.make_node("MatMul", ["X", "W"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 1, 2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3, 2, 2])],
            [numpy_helper.from_array(weight, "W")],
        )
        return model, {"X": x}
    if name.startswith("Conv"):
        value = np.arange(1, 26, dtype=np.float32).reshape(1, 1, 5, 5) / 10.0
        weight = np.array([[1, -1], [0.5, 2]], dtype=np.float32).reshape(1, 1, 2, 2)
        attributes = {}
        if name == "ConvAutoPadNotSet":
            attributes = {"auto_pad": "NOTSET", "pads": [1, 0, 0, 1]}
        elif name == "ConvAutoPadSameUpper":
            attributes = {"auto_pad": "SAME_UPPER", "strides": [2, 2]}
        elif name == "ConvAutoPadSameLower":
            attributes = {"auto_pad": "SAME_LOWER", "strides": [2, 2]}
        elif name == "ConvAutoPadValid":
            attributes = {"auto_pad": "VALID"}
        elif name == "ConvStrides":
            attributes = {"auto_pad": "NOTSET", "strides": [2, 2]}
        elif name == "ConvDilations":
            attributes = {"auto_pad": "NOTSET", "dilations": [2, 2]}
        elif name == "ConvDepthwise":
            value = np.arange(1, 51, dtype=np.float32).reshape(1, 2, 5, 5) / 10.0
            weight = np.array([
                [[1, 0], [0, -1]],
                [[2, 1], [-1, 0.5]],
            ], dtype=np.float32).reshape(2, 1, 2, 2)
            attributes = {"auto_pad": "NOTSET", "group": 2, "pads": [1, 1, 1, 1]}
        else:
            raise ValueError("unknown parity operator: " + name)
        output_shape = [1, weight.shape[0], 5, 5]
        if name in ("ConvAutoPadSameUpper", "ConvAutoPadSameLower"):
            output_shape = [1, weight.shape[0], 3, 3]
        elif name == "ConvAutoPadValid":
            output_shape = [1, weight.shape[0], 4, 4]
        elif name == "ConvStrides":
            output_shape = [1, weight.shape[0], 2, 2]
        elif name == "ConvDilations":
            output_shape = [1, weight.shape[0], 3, 3]
        elif name == "ConvDepthwise":
            output_shape = [1, weight.shape[0], 6, 6]
        model = make_model(
            [helper.make_node("Conv", ["X", "W"], ["Y"], **attributes)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, list(value.shape))],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, output_shape)],
            [numpy_helper.from_array(weight, "W")],
        )
        return model, {"X": value}
    if name in ("MaxPoolStridesPads", "AveragePoolExcludePad", "AveragePoolIncludePad"):
        value = np.arange(1, 10, dtype=np.float32).reshape(1, 1, 3, 3)
        attributes = {"kernel_shape": [2, 2], "pads": [1, 0, 0, 1], "strides": [2, 1]}
        if name == "MaxPoolStridesPads":
            op_type = "MaxPool"
        else:
            op_type = "AveragePool"
            attributes["count_include_pad"] = 1 if name == "AveragePoolIncludePad" else 0
        model = make_model(
            [helper.make_node(op_type, ["X"], ["Y"], **attributes)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 1, 3, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 1, 2, 3])],
        )
        return model, {"X": value}
    if name == "GlobalAveragePool":
        value = np.arange(1, 13, dtype=np.float32).reshape(1, 2, 2, 3)
        model = make_model(
            [helper.make_node("GlobalAveragePool", ["X"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 2, 2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 2, 1, 1])],
        )
        return model, {"X": value}
    if name == "BatchNormalizationEpsilon":
        value = np.array([1, 3, 10, 14], dtype=np.float32).reshape(1, 2, 1, 2)
        scale = np.array([2, 0.5], dtype=np.float32)
        bias = np.array([1, -1], dtype=np.float32)
        mean = np.array([1, 12], dtype=np.float32)
        variance = np.array([4, 4], dtype=np.float32)
        model = make_model(
            [helper.make_node("BatchNormalization", ["X", "scale", "bias", "mean", "variance"], ["Y"], epsilon=0.001)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 2, 1, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 2, 1, 2])],
            [
                numpy_helper.from_array(scale, "scale"),
                numpy_helper.from_array(bias, "bias"),
                numpy_helper.from_array(mean, "mean"),
                numpy_helper.from_array(variance, "variance"),
            ],
        )
        return model, {"X": value}
    if name in ("PadAttributes", "PadInitializers"):
        value = np.array([[1, 2]], dtype=np.float32)
        pads = np.array([1, 0, 2, 1], dtype=np.int64)
        constant = np.array(0.5, dtype=np.float32)
        if name == "PadAttributes":
            node = helper.make_node("Pad", ["X"], ["Y"], pads=pads.tolist(), value=0.5, mode="constant")
            initializers = []
            opset = 10
        else:
            node = helper.make_node("Pad", ["X", "pads", "constant"], ["Y"], mode="constant")
            initializers = [numpy_helper.from_array(pads, "pads"), numpy_helper.from_array(constant, "constant")]
            opset = 13
        model = make_model(
            [node],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [4, 3])],
            initializers,
            opset=opset,
        )
        return model, {"X": value}
    if name in ("Add", "Sub", "Mul", "Div"):
        left = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        right = np.array([2, 4, 5] if name == "Div" else [10, 20, 30], dtype=np.float32)
        model = make_model(
            [helper.make_node(name, ["A", "B"], ["Y"])],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 3]), helper.make_tensor_value_info("B", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"A": left, "B": right}
    if name == "HalfInitializer":
        left = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        half = np.array([1.1, -2.5, 2**-14], dtype=np.float16)
        model = make_model(
            [
                helper.make_node("Cast", ["H"], ["B"], to=TensorProto.FLOAT),
                helper.make_node("Add", ["A", "B"], ["Y"]),
            ],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
            [numpy_helper.from_array(half, "H")],
        )
        return model, {"A": left}
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
        axes = np.array([-1], dtype=np.int64)
        model = make_model(
            [helper.make_node("Unsqueeze", ["X", "axes"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 1])],
            [numpy_helper.from_array(axes, "axes")],
        )
        return model, {"X": value}
    if name == "Squeeze":
        value = np.array([1, 2], dtype=np.float32).reshape(1, 2, 1)
        axes = np.array([-1], dtype=np.int64)
        model = make_model(
            [helper.make_node("Squeeze", ["X", "axes"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 2, 1])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 2])],
            [numpy_helper.from_array(axes, "axes")],
        )
        return model, {"X": value}
    if name == "Expand":
        value = np.array([[1], [2]], dtype=np.float32)
        shape = np.array([2, 3], dtype=np.int64)
        model = make_model(
            [helper.make_node("Expand", ["X", "shape"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 1])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
            [numpy_helper.from_array(shape, "shape")],
        )
        return model, {"X": value}
    if name == "Shape":
        value = np.zeros((2, 3, 4), dtype=np.float32)
        model = make_model(
            [helper.make_node("Shape", ["X"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3, 4])],
            [helper.make_tensor_value_info("Y", TensorProto.INT64, [3])],
        )
        return model, {"X": value}
    if name == "Slice":
        value = np.arange(12, dtype=np.float32).reshape(3, 4)
        starts = np.array([0, 1], dtype=np.int64)
        ends = np.array([3, 4], dtype=np.int64)
        axes = np.array([0, 1], dtype=np.int64)
        steps = np.array([1, 2], dtype=np.int64)
        model = make_model(
            [helper.make_node("Slice", ["X", "starts", "ends", "axes", "steps"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [3, 4])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [3, 2])],
            [numpy_helper.from_array(starts, "starts"), numpy_helper.from_array(ends, "ends"), numpy_helper.from_array(axes, "axes"), numpy_helper.from_array(steps, "steps")],
        )
        return model, {"X": value}
    if name == "Split":
        value = np.array([[1, 2, 3, 4], [5, 6, 7, 8]], dtype=np.float32)
        split = np.array([2, 2], dtype=np.int64)
        model = make_model(
            [helper.make_node("Split", ["X", "split"], ["Y1", "Y2"], axis=1)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 4])],
            [helper.make_tensor_value_info("Y1", TensorProto.FLOAT, [2, 2]), helper.make_tensor_value_info("Y2", TensorProto.FLOAT, [2, 2])],
            [numpy_helper.from_array(split, "split")],
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
    if name in ("Equal", "Greater"):
        left = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        right = np.array([1, 2, 4] if name == "Equal" else [0, 2, 4], dtype=np.float32)
        model = make_model(
            [helper.make_node(name, ["A", "B"], ["Y"])],
            [helper.make_tensor_value_info("A", TensorProto.FLOAT, [2, 3]), helper.make_tensor_value_info("B", TensorProto.FLOAT, [3])],
            [helper.make_tensor_value_info("Y", TensorProto.BOOL, [2, 3])],
        )
        return model, {"A": left, "B": right}
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
        attributes = {"axis": 0} if name == "Softmax" else {}
        model = make_model(
            [helper.make_node(name, ["X"], ["Y"], **attributes)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"X": value}
    if name == "LayerNormalization":
        value = np.array([
            -1, 0, 1, 2, 2, 1, 0, -1,
            0.5, -0.5, 1.5, -1.5, 3, 2, 1, 0,
            -2, -1, 0, 1, 1.5, 0.5, -0.5, -1.5,
        ], dtype=np.float32).reshape(2, 3, 4)
        scale = np.array([1, 0.5, 2, -1], dtype=np.float32)
        bias = np.array([0.1, -0.2, 0.3, 0.4], dtype=np.float32)
        model = make_model(
            [helper.make_node("LayerNormalization", ["X", "scale", "bias"], ["Y"], axis=-1, epsilon=1e-5)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3, 4])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3, 4])],
            [numpy_helper.from_array(scale, "scale"), numpy_helper.from_array(bias, "bias")],
            opset=17,
        )
        return model, {"X": value}
    if name == "LayerNormalizationAxis1":
        value = np.array([
            -1, 0, 1, 2, 2, 1, 0, -1,
            0.5, -0.5, 1.5, -1.5, 3, 2, 1, 0,
            -2, -1, 0, 1, 1.5, 0.5, -0.5, -1.5,
        ], dtype=np.float32).reshape(2, 3, 4)
        scale = np.array([
            1, 0.5, 2, -1, 0.75, 1.25, -0.5, 0.25,
            1.5, -0.25, 0.5, 2,
        ], dtype=np.float32).reshape(3, 4)
        bias = np.array([
            0.1, -0.2, 0.3, 0.4, -0.1, 0.2, -0.3, 0.5,
            0.25, -0.4, 0.15, 0.05,
        ], dtype=np.float32).reshape(3, 4)
        model = make_model(
            [helper.make_node("LayerNormalization", ["X", "scale", "bias"], ["Y"], axis=1, epsilon=1e-5)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3, 4])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3, 4])],
            [numpy_helper.from_array(scale, "scale"), numpy_helper.from_array(bias, "bias")],
            opset=17,
        )
        return model, {"X": value}
    if name == "Gelu":
        value = np.array([[-2, -1, 0], [0.5, 1, 2]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Gelu", ["X"], ["Y"], approximate="none")],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
            opset=20,
        )
        return model, {"X": value}
    if name == "Erf":
        value = np.array([[-2, -1, 0], [0.5, 1, 2]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Erf", ["X"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"X": value}
    if name == "Sqrt":
        value = np.array([[0, 1, 4], [9, 16, 25]], dtype=np.float32)
        model = make_model(
            [helper.make_node("Sqrt", ["X"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
        )
        return model, {"X": value}
    if name == "Pow":
        value = np.array([[1, 2, 3], [4, 5, 6]], dtype=np.float32)
        exponent = np.array([1, 2, 0.5], dtype=np.float32)
        model = make_model(
            [helper.make_node("Pow", ["X", "exponent"], ["Y"])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 3])],
            [numpy_helper.from_array(exponent, "exponent")],
        )
        return model, {"X": value}
    if name in ("ReduceMean", "ReduceMeanMultiAxes", "ReduceMeanNoKeepdims", "ReduceMeanInitializer"):
        if name == "ReduceMean":
            value = np.arange(1, 13, dtype=np.float32).reshape(2, 3, 2)
            axes = np.array([-1], dtype=np.int64)
            output_shape = [2, 3, 1]
            keepdims = 1
        else:
            value = np.arange(1, 25, dtype=np.float32).reshape(2, 3, 4)
            axes = np.array([0, 2], dtype=np.int64)
            output_shape = [1, 3, 1] if name == "ReduceMeanMultiAxes" else [3]
            keepdims = 1 if name == "ReduceMeanMultiAxes" else 0
        if name == "ReduceMeanInitializer":
            inputs = ["X", "axes"]
            attributes = {"keepdims": keepdims}
            initializers = [numpy_helper.from_array(axes, "axes")]
            opset = 18
        else:
            inputs = ["X"]
            attributes = {"axes": axes.tolist(), "keepdims": keepdims}
            initializers = []
            # ReduceMean used the axes attribute before opset 13. Keep these
            # rows on that schema so the parity harness covers the attribute
            # form as well as the initializer-input form above.
            opset = 12
        model = make_model(
            [helper.make_node("ReduceMean", inputs, ["Y"], **attributes)],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, list(value.shape))],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, output_shape)],
            initializers,
            opset=opset,
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
        value = np.arange(12, dtype=np.float32).reshape(2, 3, 2)
        model = make_model(
            [helper.make_node("Transpose", ["X"], ["Y"], perm=[2, 0, 1])],
            [helper.make_tensor_value_info("X", TensorProto.FLOAT, [2, 3, 2])],
            [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [2, 2, 3])],
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


def encoder_model():
    x = np.array([
        0.25, -0.5, 1.0, 0.75,
        -1.25, 0.5, 0.25, 1.5,
    ], dtype=np.float32).reshape(1, 2, 4)

    def matrix(rows, cols, offset):
        return ((np.arange(rows * cols, dtype=np.float32).reshape(rows, cols) + offset) / 17.0)

    wq, wk, wv, wo = (matrix(4, 4, offset) for offset in (1, 3, 5, 7))
    bq = np.array([0.01, -0.02, 0.03, -0.04], dtype=np.float32)
    bk = np.array([-0.03, 0.02, -0.01, 0.04], dtype=np.float32)
    bv = np.array([0.02, 0.01, -0.04, 0.03], dtype=np.float32)
    bo = np.array([0.01, 0.02, -0.02, -0.01], dtype=np.float32)
    w1 = matrix(4, 6, 9)
    b1 = np.array([0.01, -0.01, 0.02, -0.02, 0.03, -0.03], dtype=np.float32)
    w2 = matrix(6, 4, 11)
    b2 = np.array([0.03, -0.02, 0.01, -0.04], dtype=np.float32)
    gamma1 = np.array([1.0, 0.9, 1.1, 0.8], dtype=np.float32)
    beta1 = np.array([0.0, 0.1, -0.1, 0.05], dtype=np.float32)
    gamma2 = np.array([0.95, 1.05, 0.85, 1.15], dtype=np.float32)
    beta2 = np.array([0.02, -0.03, 0.04, -0.05], dtype=np.float32)
    q_shape = np.array([1, 2, 2, 2], dtype=np.int64)
    context_shape = np.array([1, 2, 4], dtype=np.int64)
    scale = np.array(np.sqrt(2.0), dtype=np.float32)

    nodes = [
        helper.make_node("MatMul", ["X", "Wq"], ["Q"]),
        helper.make_node("Add", ["Q", "bq"], ["Qb"]),
        helper.make_node("Reshape", ["Qb", "q_shape"], ["Q4"]),
        helper.make_node("Transpose", ["Q4"], ["Qh"], perm=[0, 2, 1, 3]),
        helper.make_node("MatMul", ["X", "Wk"], ["K"]),
        helper.make_node("Add", ["K", "bk"], ["Kb"]),
        helper.make_node("Reshape", ["Kb", "q_shape"], ["K4"]),
        helper.make_node("Transpose", ["K4"], ["Kh"], perm=[0, 2, 3, 1]),
        helper.make_node("MatMul", ["Qh", "Kh"], ["Scores"]),
        helper.make_node("Div", ["Scores", "scale"], ["Scaled"]),
        helper.make_node("Softmax", ["Scaled"], ["Prob"], axis=-1),
        helper.make_node("MatMul", ["X", "Wv"], ["V"]),
        helper.make_node("Add", ["V", "bv"], ["Vb"]),
        helper.make_node("Reshape", ["Vb", "q_shape"], ["V4"]),
        helper.make_node("Transpose", ["V4"], ["Vh"], perm=[0, 2, 1, 3]),
        helper.make_node("MatMul", ["Prob", "Vh"], ["Context"]),
        helper.make_node("Transpose", ["Context"], ["ContextT"], perm=[0, 2, 1, 3]),
        helper.make_node("Reshape", ["ContextT", "context_shape"], ["Context2"]),
        helper.make_node("MatMul", ["Context2", "Wo"], ["Projected"]),
        helper.make_node("Add", ["Projected", "bo"], ["Attn"]),
        helper.make_node("Add", ["X", "Attn"], ["R1"]),
        helper.make_node("LayerNormalization", ["R1", "gamma1", "beta1"], ["N1"], axis=-1, epsilon=1e-5),
        helper.make_node("MatMul", ["N1", "W1"], ["H1"]),
        helper.make_node("Add", ["H1", "b1"], ["H1b"]),
        helper.make_node("Gelu", ["H1b"], ["H2"], approximate="none"),
        helper.make_node("MatMul", ["H2", "W2"], ["F"]),
        helper.make_node("Add", ["F", "b2"], ["Fb"]),
        helper.make_node("Add", ["N1", "Fb"], ["R2"]),
        helper.make_node("LayerNormalization", ["R2", "gamma2", "beta2"], ["Y"], axis=-1, epsilon=1e-5),
    ]
    initializers = [
        numpy_helper.from_array(value, name)
        for name, value in [
            ("Wq", wq), ("Wk", wk), ("Wv", wv), ("Wo", wo),
            ("bq", bq), ("bk", bk), ("bv", bv), ("bo", bo),
            ("W1", w1), ("b1", b1), ("W2", w2), ("b2", b2),
            ("gamma1", gamma1), ("beta1", beta1), ("gamma2", gamma2), ("beta2", beta2),
            ("q_shape", q_shape), ("context_shape", context_shape), ("scale", scale),
        ]
    ]
    model = make_model(
        nodes,
        [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 2, 4])],
        [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 2, 4])],
        initializers,
        opset=20,
    )
    return model, {"X": x}


def cnn_model():
    x = ((np.arange(64, dtype=np.float32).reshape(1, 1, 8, 8) - 32.0) / 32.0)
    w1 = ((np.arange(18, dtype=np.float32).reshape(2, 1, 3, 3) - 8.0) / 17.0)
    b1 = np.array([0.1, -0.2], dtype=np.float32)
    scale1 = np.array([1.0, 0.75], dtype=np.float32)
    bias1 = np.array([0.1, -0.15], dtype=np.float32)
    mean1 = np.array([0.0, 0.25], dtype=np.float32)
    variance1 = np.array([1.0, 0.5], dtype=np.float32)
    w2 = ((np.arange(72, dtype=np.float32).reshape(4, 2, 3, 3) - 36.0) / 29.0)
    b2 = np.array([0.05, -0.1, 0.15, -0.2], dtype=np.float32)
    wg = ((np.arange(40, dtype=np.float32).reshape(4, 10) - 20.0) / 37.0)
    bg = np.array([0.1, -0.05, 0.2, -0.15, 0.3, -0.25, 0.4, -0.35, 0.5, -0.45], dtype=np.float32)
    nodes = [
        helper.make_node("Conv", ["X", "W1", "B1"], ["C1"], pads=[1, 1, 1, 1]),
        helper.make_node("BatchNormalization", ["C1", "scale1", "bias1", "mean1", "variance1"], ["N1"], epsilon=0.001),
        helper.make_node("Relu", ["N1"], ["R1"]),
        helper.make_node("MaxPool", ["R1"], ["P1"], kernel_shape=[2, 2], strides=[2, 2]),
        helper.make_node("Conv", ["P1", "W2", "B2"], ["C2"], pads=[1, 1, 1, 1]),
        helper.make_node("Relu", ["C2"], ["R2"]),
        helper.make_node("GlobalAveragePool", ["R2"], ["G"]),
        helper.make_node("Flatten", ["G"], ["F"], axis=1),
        helper.make_node("Gemm", ["F", "WG", "BG"], ["L"]),
        helper.make_node("Softmax", ["L"], ["Y"], axis=1),
    ]
    initializers = [
        numpy_helper.from_array(value, name)
        for name, value in [
            ("W1", w1), ("B1", b1),
            ("scale1", scale1), ("bias1", bias1), ("mean1", mean1), ("variance1", variance1),
            ("W2", w2), ("B2", b2), ("WG", wg), ("BG", bg),
        ]
    ]
    model = make_model(
        nodes,
        [helper.make_tensor_value_info("X", TensorProto.FLOAT, [1, 1, 8, 8])],
        [helper.make_tensor_value_info("Y", TensorProto.FLOAT, [1, 10])],
        initializers,
        opset=13,
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
    if mode == "encoder":
        model, feed = encoder_model()
        with open(sys.argv[2], "wb") as handle:
            handle.write(model.SerializeToString())
        run_model(model, feed)
        return
    if mode == "cnn":
        model, feed = cnn_model()
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
