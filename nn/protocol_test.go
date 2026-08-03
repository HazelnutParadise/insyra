package nn_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
	"github.com/HazelnutParadise/insyra/nn"
	"google.golang.org/protobuf/encoding/protowire"
)

var _ ml.Model = (*nn.BoundRegressor)(nil)
var _ ml.ProbaModel = (*nn.BoundClassifier)(nil)

func TestBoundAdaptersSatisfyMLConformance(t *testing.T) {
	regressorModel, err := nn.LoadONNX(bytes.NewReader(protocolIdentityModel("X", []int64{-1, 1}, "Y", []int64{-1, 1})))
	if err != nil {
		t.Fatalf("load regressor: %v", err)
	}
	regressor, err := nn.BindRegressor(regressorModel, "X", []string{"value"})
	if err != nil {
		t.Fatalf("bind regressor: %v", err)
	}
	x := insyra.NewDataTable(
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0).SetName("value"),
	)
	mltest.RunConformance(t, regressor, x, nil)

	classifierModel, err := nn.LoadONNX(bytes.NewReader(protocolIdentityModel("X", []int64{-1, 2}, "probabilities", []int64{-1, 2})))
	if err != nil {
		t.Fatalf("load classifier: %v", err)
	}
	classifier, err := nn.BindClassifier(classifierModel, "X", []string{"left", "right"}, insyra.NewDataList("cold", "warm"))
	if err != nil {
		t.Fatalf("bind classifier: %v", err)
	}
	classifierInput := insyra.NewDataTable(
		insyra.NewDataList(0.9, 0.1, 0.2, 0.8).SetName("left"),
		insyra.NewDataList(0.1, 0.9, 0.8, 0.2).SetName("right"),
	)
	mltest.RunConformance(t, classifier, classifierInput, insyra.NewDataList("cold", "warm", "warm", "cold"))
}

func TestBindAdaptersRefuseMismatchesByName(t *testing.T) {
	model, err := nn.LoadONNX(bytes.NewReader(protocolIdentityModel("features", []int64{-1, 2}, "prediction", []int64{-1, 1})))
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	if _, err := nn.BindRegressor(model, "unknown", []string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown input error = %v, want input name", err)
	}
	if _, err := nn.BindRegressor(model, "features", []string{"a"}); err == nil || !strings.Contains(err.Error(), "features") {
		t.Fatalf("feature width error = %v, want input name", err)
	}

	classifierModel, err := nn.LoadONNX(bytes.NewReader(protocolIdentityModel("features", []int64{-1, 2}, "probabilities", []int64{-1, 2})))
	if err != nil {
		t.Fatalf("load classifier model: %v", err)
	}
	if _, err := nn.BindClassifier(classifierModel, "features", []string{"a", "b"}, insyra.NewDataList("zero", "one", "two")); err == nil || !strings.Contains(err.Error(), "probabilities") {
		t.Fatalf("class width error = %v, want probability output name", err)
	}
}

func TestBoundClassifierRefusesInvalidProbabilities(t *testing.T) {
	model, err := nn.LoadONNX(bytes.NewReader(protocolIdentityModel("features", []int64{-1, 2}, "probabilities", []int64{-1, 2})))
	if err != nil {
		t.Fatalf("load classifier model: %v", err)
	}
	classifier, err := nn.BindClassifier(model, "features", []string{"a", "b"}, insyra.NewDataList("zero", "one"))
	if err != nil {
		t.Fatalf("bind classifier: %v", err)
	}
	for name, values := range map[string][]float64{
		"negative": {-1, 2},
		"nan":      {math.NaN(), 1},
		"infinite": {math.Inf(1), 1},
	} {
		t.Run(name, func(t *testing.T) {
			input := insyra.NewDataTable(
				insyra.NewDataList(values[0]).SetName("a"),
				insyra.NewDataList(values[1]).SetName("b"),
			)
			if _, err := classifier.PredictProba(input); err == nil || !strings.Contains(err.Error(), "probability output") {
				t.Fatalf("PredictProba error = %v, want a named probability refusal", err)
			}
		})
	}
}

func protocolIdentityModel(inputName string, inputShape []int64, outputName string, outputShape []int64) []byte {
	node := appendProtocolField(nil, 1, protowire.BytesType, []byte(inputName))
	node = appendProtocolField(node, 2, protowire.BytesType, []byte(outputName))
	node = appendProtocolField(node, 4, protowire.BytesType, []byte("Identity"))
	graph := appendProtocolField(nil, 1, protowire.BytesType, node)
	graph = appendProtocolField(graph, 11, protowire.BytesType, protocolValueInfo(inputName, 1, inputShape))
	graph = appendProtocolField(graph, 12, protowire.BytesType, protocolValueInfo(outputName, 1, outputShape))
	model := appendProtocolVarint(nil, 1, 9)
	model = appendProtocolField(model, 7, protowire.BytesType, graph)
	model = appendProtocolField(model, 8, protowire.BytesType, appendProtocolVarint(nil, 2, 13))
	return model
}

func protocolValueInfo(name string, elemType int32, shape []int64) []byte {
	tensorType := appendProtocolVarint(nil, 1, uint64(elemType))
	shapeMessage := []byte(nil)
	for _, dimension := range shape {
		dimensionMessage := []byte(nil)
		if dimension < 0 {
			dimensionMessage = appendProtocolField(dimensionMessage, 2, protowire.BytesType, []byte("dynamic"))
		} else {
			dimensionMessage = appendProtocolVarint(dimensionMessage, 1, uint64(dimension))
		}
		shapeMessage = appendProtocolField(shapeMessage, 1, protowire.BytesType, dimensionMessage)
	}
	tensorType = appendProtocolField(tensorType, 2, protowire.BytesType, shapeMessage)
	typeMessage := appendProtocolField(nil, 1, protowire.BytesType, tensorType)
	valueInfo := appendProtocolField(nil, 1, protowire.BytesType, []byte(name))
	return appendProtocolField(valueInfo, 2, protowire.BytesType, typeMessage)
}

func appendProtocolVarint(out []byte, number int, value uint64) []byte {
	out = protowire.AppendTag(out, protowire.Number(number), protowire.VarintType)
	return protowire.AppendVarint(out, value)
}

func appendProtocolField(out []byte, number int, typ protowire.Type, value []byte) []byte {
	out = protowire.AppendTag(out, protowire.Number(number), typ)
	if typ == protowire.BytesType {
		return protowire.AppendBytes(out, value)
	}
	return append(out, value...)
}
