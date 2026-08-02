package dl

import (
	"fmt"
	"math"
)

func linearRegressor(input *Tensor, attributes map[string]protoAttribute) (*Tensor, error) {
	if err := requireFloat32(input, "linear regressor input"); err != nil {
		return nil, err
	}
	if len(input.shape) != 2 {
		return nil, fmt.Errorf("linear regressor requires a 2-D input, got %v", input.shape)
	}
	coefficients, err := mlAttrFloats(attributes, "coefficients", true)
	if err != nil {
		return nil, err
	}
	intercepts, err := mlAttrFloats(attributes, "intercepts", false)
	if err != nil {
		return nil, err
	}
	targets := 1
	if value, present := attributes["targets"]; present {
		if !value.hasInt || value.intValue <= 0 || value.intValue > int64(maxInt()) {
			return nil, fmt.Errorf("attribute targets is not a positive integer")
		}
		targets = int(value.intValue)
	}
	if input.shape[1] == 0 {
		return nil, fmt.Errorf("linear regressor requires at least one feature")
	}
	expectedCoefficients, err := checkedProduct(input.shape[1], targets, "linear regressor coefficient count")
	if err != nil {
		return nil, err
	}
	if len(coefficients) != expectedCoefficients {
		return nil, fmt.Errorf("linear regressor has %d coefficients, want %d", len(coefficients), expectedCoefficients)
	}
	if len(intercepts) != 0 && len(intercepts) != targets {
		return nil, fmt.Errorf("linear regressor has %d intercepts, want %d", len(intercepts), targets)
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], targets})
	if err != nil {
		return nil, err
	}
	for row := 0; row < input.shape[0]; row++ {
		for target := 0; target < targets; target++ {
			score := float32(0)
			if len(intercepts) > 0 {
				score = intercepts[target]
			}
			for feature := 0; feature < input.shape[1]; feature++ {
				score += input.data[row*input.shape[1]+feature] * coefficients[target*input.shape[1]+feature]
			}
			result.data[row*targets+target] = score
		}
	}
	return applyMLPostTransform(result, mlAttrString(attributes, "post_transform", "NONE"))
}

func linearClassifier(input *Tensor, attributes map[string]protoAttribute) (*Tensor, *Tensor, error) {
	if err := requireFloat32(input, "linear classifier input"); err != nil {
		return nil, nil, err
	}
	if len(input.shape) != 2 {
		return nil, nil, fmt.Errorf("linear classifier requires a 2-D input, got %v", input.shape)
	}
	classLabels, labelType, err := mlClassLabels(attributes)
	if err != nil {
		return nil, nil, err
	}
	if len(classLabels) == 0 {
		return nil, nil, fmt.Errorf("linear classifier has no class labels")
	}
	coefficients, err := mlAttrFloats(attributes, "coefficients", true)
	if err != nil {
		return nil, nil, err
	}
	intercepts, err := mlAttrFloats(attributes, "intercepts", false)
	if err != nil {
		return nil, nil, err
	}
	postTransform, err := classifierPostTransform(attributes)
	if err != nil {
		return nil, nil, err
	}
	classCount := len(classLabels)
	features := input.shape[1]
	binaryScore := classCount == 2 && len(coefficients) == features
	scoreCount := classCount
	if binaryScore {
		scoreCount = 1
	}
	if len(coefficients) != scoreCount*features {
		return nil, nil, fmt.Errorf("linear classifier has %d coefficients, want %d", len(coefficients), scoreCount*features)
	}
	if len(intercepts) != 0 && len(intercepts) != scoreCount {
		return nil, nil, fmt.Errorf("linear classifier has %d intercepts, want %d", len(intercepts), scoreCount)
	}
	probabilities, err := newZeroFloat32Tensor([]int{input.shape[0], classCount})
	if err != nil {
		return nil, nil, err
	}
	labels, err := newClassLabelTensor(labelType, classLabels, input.shape[0])
	if err != nil {
		return nil, nil, err
	}
	for row := 0; row < input.shape[0]; row++ {
		scores := make([]float32, scoreCount)
		for scoreIndex := range scores {
			score := float32(0)
			if len(intercepts) > 0 {
				score = intercepts[scoreIndex]
			}
			for feature := 0; feature < features; feature++ {
				score += input.data[row*features+feature] * coefficients[scoreIndex*features+feature]
			}
			scores[scoreIndex] = score
		}
		if binaryScore {
			positive := scores[0]
			if postTransform == "LOGISTIC" {
				positive = sigmoidScalar(positive)
			}
			probabilities.data[row*classCount] = 1 - positive
			probabilities.data[row*classCount+1] = positive
			setClassLabel(labels, row, classLabels, labelType, boolToInt(positive > 0.5))
			continue
		}
		for scoreIndex, score := range scores {
			if postTransform == "LOGISTIC" {
				score = sigmoidScalar(score)
			}
			probabilities.data[row*classCount+scoreIndex] = score
		}
		if postTransform == "SOFTMAX" || postTransform == "SOFTMAX_ZERO" {
			softmaxRow(probabilities.data[row*classCount : (row+1)*classCount])
		}
		best := 0
		for classIndex := 1; classIndex < classCount; classIndex++ {
			if probabilities.data[row*classCount+classIndex] > probabilities.data[row*classCount+best] {
				best = classIndex
			}
		}
		setClassLabel(labels, row, classLabels, labelType, best)
	}
	return labels, probabilities, nil
}

func scaler(input *Tensor, attributes map[string]protoAttribute) (*Tensor, error) {
	if err := requireFloat32(input, "scaler input"); err != nil {
		return nil, err
	}
	offset, err := mlAttrFloats(attributes, "offset", true)
	if err != nil {
		return nil, err
	}
	scale, err := mlAttrFloats(attributes, "scale", true)
	if err != nil {
		return nil, err
	}
	if len(offset) != len(scale) {
		return nil, fmt.Errorf("scaler offset and scale lengths differ: %d and %d", len(offset), len(scale))
	}
	if len(input.shape) == 0 || input.shape[len(input.shape)-1] != len(offset) {
		return nil, fmt.Errorf("scaler expects last dimension %d, got shape %v", len(offset), input.shape)
	}
	result, err := copyTensor(input)
	if err != nil {
		return nil, err
	}
	for index := range result.data {
		feature := index % len(offset)
		result.data[index] = (result.data[index] - offset[feature]) * scale[feature]
	}
	return result, nil
}

func oneHotEncoder(input *Tensor, attributes map[string]protoAttribute) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("one-hot encoder input is nil")
	}
	if len(input.shape) != 1 {
		return nil, fmt.Errorf("one-hot encoder requires a rank-1 input, got %v", input.shape)
	}
	strings, hasStrings := attributes["cats_strings"]
	ints, hasInts := attributes["cats_int64s"]
	if hasStrings == hasInts {
		return nil, fmt.Errorf("one-hot encoder needs exactly one category attribute")
	}
	categoryCount := len(strings.strings)
	if hasInts {
		categoryCount = len(ints.ints)
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], categoryCount})
	if err != nil {
		return nil, err
	}
	for row := 0; row < input.shape[0]; row++ {
		category := -1
		if hasStrings {
			if input.dtype != DTypeString {
				return nil, fmt.Errorf("one-hot encoder categories are strings, input has dtype %s", dtypeName(input.dtype))
			}
			for index, value := range strings.strings {
				if string(value) == input.stringData[row] {
					category = index
					break
				}
			}
		} else {
			if input.dtype != DTypeInt64 {
				return nil, fmt.Errorf("one-hot encoder categories are int64, input has dtype %s", dtypeName(input.dtype))
			}
			for index, value := range ints.ints {
				if value == input.int64Data[row] {
					category = index
					break
				}
			}
		}
		if category >= 0 {
			result.data[row*categoryCount+category] = 1
		}
	}
	return result, nil
}

func labelEncoder(input *Tensor, attributes map[string]protoAttribute) (*Tensor, error) {
	if input == nil {
		return nil, fmt.Errorf("label encoder input is nil")
	}
	keysStrings, hasStrings := attributes["keys_strings"]
	keysInts, hasInts := attributes["keys_int64s"]
	keysFloats, hasFloats := attributes["keys_floats"]
	keyKinds := boolToInt(hasStrings) + boolToInt(hasInts) + boolToInt(hasFloats)
	if keyKinds != 1 {
		return nil, fmt.Errorf("label encoder needs exactly one key attribute")
	}
	values, err := mlAttrInts(attributes, "values_int64s", true)
	if err != nil {
		return nil, err
	}
	if (hasStrings && len(keysStrings.strings) != len(values)) || (hasInts && len(keysInts.ints) != len(values)) || (hasFloats && len(keysFloats.floats) != len(values)) {
		return nil, fmt.Errorf("label encoder key and value lengths differ")
	}
	defaultValue := int64(0)
	if value, present := attributes["default_int64"]; present {
		if !value.hasInt {
			return nil, fmt.Errorf("attribute default_int64 is not an integer")
		}
		defaultValue = value.intValue
	}
	result, err := newInt64Tensor(input.shape, make([]int64, input.Len()))
	if err != nil {
		return nil, err
	}
	for index := range result.int64Data {
		result.int64Data[index] = defaultValue
		for keyIndex, encoded := range values {
			match := false
			switch {
			case hasStrings:
				if input.dtype != DTypeString {
					return nil, fmt.Errorf("label encoder keys are strings, input has dtype %s", dtypeName(input.dtype))
				}
				match = string(keysStrings.strings[keyIndex]) == input.stringData[index]
			case hasInts:
				if input.dtype != DTypeInt64 {
					return nil, fmt.Errorf("label encoder keys are int64, input has dtype %s", dtypeName(input.dtype))
				}
				match = keysInts.ints[keyIndex] == input.int64Data[index]
			case hasFloats:
				if input.dtype != DTypeFloat32 {
					return nil, fmt.Errorf("label encoder keys are floats, input has dtype %s", dtypeName(input.dtype))
				}
				match = keysFloats.floats[keyIndex] == input.data[index]
			}
			if match {
				result.int64Data[index] = encoded
				break
			}
		}
	}
	return result, nil
}

func treeEnsembleRegressor(input *Tensor, attributes map[string]protoAttribute) (*Tensor, error) {
	if err := requireFloat32(input, "tree ensemble regressor input"); err != nil {
		return nil, err
	}
	if len(input.shape) != 2 {
		return nil, fmt.Errorf("tree ensemble regressor requires a 2-D input, got %v", input.shape)
	}
	forest, err := buildTreeEnsemble(attributes)
	if err != nil {
		return nil, err
	}
	targets := 1
	if value, present := attributes["n_targets"]; present {
		if !value.hasInt || value.intValue <= 0 || value.intValue > int64(maxInt()) {
			return nil, fmt.Errorf("attribute n_targets is not a positive integer")
		}
		targets = int(value.intValue)
	}
	targetTreeIDs, err := mlAttrInts(attributes, "target_treeids", true)
	if err != nil {
		return nil, err
	}
	targetNodeIDs, err := mlAttrInts(attributes, "target_nodeids", true)
	if err != nil {
		return nil, err
	}
	targetIDs, err := mlAttrInts(attributes, "target_ids", true)
	if err != nil {
		return nil, err
	}
	targetWeights, err := mlAttrFloats(attributes, "target_weights", true)
	if err != nil {
		return nil, err
	}
	if len(targetTreeIDs) != len(targetNodeIDs) || len(targetIDs) != len(targetTreeIDs) || len(targetWeights) != len(targetTreeIDs) {
		return nil, fmt.Errorf("tree ensemble regressor target attributes have different lengths")
	}
	if len(targetIDs) == 0 {
		return nil, fmt.Errorf("tree ensemble regressor has no target entries")
	}
	maxTargetID := int64(-1)
	for _, targetID := range targetIDs {
		if targetID < 0 || targetID >= int64(targets) {
			return nil, fmt.Errorf("target id %d is outside %d targets", targetID, targets)
		}
		if targetID > maxTargetID {
			maxTargetID = targetID
		}
	}
	if maxTargetID+1 < int64(targets) {
		return nil, fmt.Errorf("tree ensemble regressor declares %d targets but uses only %d", targets, maxTargetID+1)
	}
	baseValues, err := mlAttrFloats(attributes, "base_values", false)
	if err != nil {
		return nil, err
	}
	if len(baseValues) > targets {
		return nil, fmt.Errorf("tree ensemble regressor has %d base values, want at most %d", len(baseValues), targets)
	}
	result, err := newZeroFloat32Tensor([]int{input.shape[0], targets})
	if err != nil {
		return nil, err
	}
	for row := 0; row < input.shape[0]; row++ {
		for target := 0; target < targets; target++ {
			if target < len(baseValues) {
				result.data[row*targets+target] = baseValues[target]
			}
		}
		rowValues := input.data[row*input.shape[1] : (row+1)*input.shape[1]]
		for _, treeID := range forest.order {
			root := forest.roots[treeID]
			node, walkErr := forest.walk(rowValues, treeID, root)
			if walkErr != nil {
				return nil, walkErr
			}
			for index := range targetTreeIDs {
				if targetTreeIDs[index] == treeID && targetNodeIDs[index] == node {
					target := targetIDs[index]
					if target < 0 || target >= int64(targets) {
						return nil, fmt.Errorf("target id %d is outside %d targets", target, targets)
					}
					result.data[row*targets+int(target)] += targetWeights[index]
				}
			}
		}
	}
	return applyMLPostTransform(result, mlAttrString(attributes, "post_transform", "NONE"))
}

func treeEnsembleClassifier(input *Tensor, attributes map[string]protoAttribute) (*Tensor, *Tensor, error) {
	if err := requireFloat32(input, "tree ensemble classifier input"); err != nil {
		return nil, nil, err
	}
	if len(input.shape) != 2 {
		return nil, nil, fmt.Errorf("tree ensemble classifier requires a 2-D input, got %v", input.shape)
	}
	forest, err := buildTreeEnsemble(attributes)
	if err != nil {
		return nil, nil, err
	}
	classLabels, labelType, err := mlClassLabels(attributes)
	if err != nil {
		return nil, nil, err
	}
	if len(classLabels) == 0 {
		return nil, nil, fmt.Errorf("tree ensemble classifier has no class labels")
	}
	classTreeIDs, err := mlAttrInts(attributes, "class_treeids", true)
	if err != nil {
		return nil, nil, err
	}
	classNodeIDs, err := mlAttrInts(attributes, "class_nodeids", true)
	if err != nil {
		return nil, nil, err
	}
	classIDs, err := mlAttrInts(attributes, "class_ids", true)
	if err != nil {
		return nil, nil, err
	}
	classWeights, err := mlAttrFloats(attributes, "class_weights", true)
	if err != nil {
		return nil, nil, err
	}
	if len(classTreeIDs) != len(classNodeIDs) || len(classIDs) != len(classTreeIDs) || len(classWeights) != len(classTreeIDs) {
		return nil, nil, fmt.Errorf("tree ensemble classifier class attributes have different lengths")
	}
	binaryScore := len(classLabels) == 2 && len(classIDs) > 0 && allZero(classIDs)
	postTransform, err := classifierPostTransform(attributes)
	if err != nil {
		return nil, nil, err
	}
	probabilities, err := newZeroFloat32Tensor([]int{input.shape[0], len(classLabels)})
	if err != nil {
		return nil, nil, err
	}
	labels, err := newClassLabelTensor(labelType, classLabels, input.shape[0])
	if err != nil {
		return nil, nil, err
	}
	baseValues, err := mlAttrFloats(attributes, "base_values", false)
	if err != nil {
		return nil, nil, err
	}
	for row := 0; row < input.shape[0]; row++ {
		scores := make([]float32, len(classLabels))
		if binaryScore && len(baseValues) > 0 {
			scores[1] = baseValues[0]
		} else {
			copy(scores, baseValues)
		}
		rowValues := input.data[row*input.shape[1] : (row+1)*input.shape[1]]
		for _, treeID := range forest.order {
			root := forest.roots[treeID]
			node, walkErr := forest.walk(rowValues, treeID, root)
			if walkErr != nil {
				return nil, nil, walkErr
			}
			for index := range classTreeIDs {
				if classTreeIDs[index] != treeID || classNodeIDs[index] != node {
					continue
				}
				classID := classIDs[index]
				if binaryScore {
					classID = 1
				}
				if classID < 0 || classID >= int64(len(scores)) {
					return nil, nil, fmt.Errorf("class id %d is outside %d classes", classID, len(scores))
				}
				scores[classID] += classWeights[index]
			}
		}
		if binaryScore {
			positive := scores[1]
			if postTransform == "LOGISTIC" {
				positive = sigmoidScalar(positive)
			}
			probabilities.data[row*2] = 1 - positive
			probabilities.data[row*2+1] = positive
			setClassLabel(labels, row, classLabels, labelType, boolToInt(positive > 0.5))
			continue
		}
		copy(probabilities.data[row*len(scores):(row+1)*len(scores)], scores)
		switch postTransform {
		case "LOGISTIC":
			for index := range scores {
				probabilities.data[row*len(scores)+index] = sigmoidScalar(scores[index])
			}
		case "SOFTMAX", "SOFTMAX_ZERO":
			softmaxRow(probabilities.data[row*len(scores) : (row+1)*len(scores)])
		}
		best := 0
		for classID := 1; classID < len(scores); classID++ {
			if probabilities.data[row*len(scores)+classID] > probabilities.data[row*len(scores)+best] {
				best = classID
			}
		}
		setClassLabel(labels, row, classLabels, labelType, best)
	}
	return labels, probabilities, nil
}

type treeEnsemble struct {
	roots map[int64]int64
	order []int64
	nodes map[treeNodeKey]treeNode
}

type treeNodeKey struct {
	tree int64
	node int64
}

type treeNode struct {
	feature     int64
	value       float32
	mode        string
	trueNode    int64
	falseNode   int64
	missingTrue bool
}

func buildTreeEnsemble(attributes map[string]protoAttribute) (treeEnsemble, error) {
	treeIDs, err := mlAttrInts(attributes, "nodes_treeids", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	nodeIDs, err := mlAttrInts(attributes, "nodes_nodeids", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	features, err := mlAttrInts(attributes, "nodes_featureids", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	values, err := mlAttrFloats(attributes, "nodes_values", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	modes, err := mlAttrStrings(attributes, "nodes_modes", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	trueNodes, err := mlAttrInts(attributes, "nodes_truenodeids", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	falseNodes, err := mlAttrInts(attributes, "nodes_falsenodeids", true)
	if err != nil {
		return treeEnsemble{}, err
	}
	missing, err := mlAttrInts(attributes, "nodes_missing_value_tracks_true", false)
	if err != nil {
		return treeEnsemble{}, err
	}
	if len(treeIDs) != len(nodeIDs) || len(features) != len(treeIDs) || len(values) != len(treeIDs) || len(modes) != len(treeIDs) || len(trueNodes) != len(treeIDs) || len(falseNodes) != len(treeIDs) {
		return treeEnsemble{}, fmt.Errorf("tree ensemble node attributes have different lengths")
	}
	if len(missing) != 0 && len(missing) != len(treeIDs) {
		return treeEnsemble{}, fmt.Errorf("tree ensemble missing-value attribute has wrong length")
	}
	forest := treeEnsemble{roots: make(map[int64]int64), nodes: make(map[treeNodeKey]treeNode, len(treeIDs))}
	for index := range treeIDs {
		key := treeNodeKey{tree: treeIDs[index], node: nodeIDs[index]}
		if _, exists := forest.nodes[key]; exists {
			return treeEnsemble{}, fmt.Errorf("tree ensemble declares node %d in tree %d more than once", nodeIDs[index], treeIDs[index])
		}
		forest.nodes[key] = treeNode{
			feature:     features[index],
			value:       values[index],
			mode:        modes[index],
			trueNode:    trueNodes[index],
			falseNode:   falseNodes[index],
			missingTrue: len(missing) > 0 && missing[index] != 0,
		}
		if _, exists := forest.roots[treeIDs[index]]; !exists {
			forest.roots[treeIDs[index]] = nodeIDs[index]
			forest.order = append(forest.order, treeIDs[index])
		}
	}
	return forest, nil
}

func (forest treeEnsemble) walk(row []float32, treeID, root int64) (int64, error) {
	nodeID := root
	for step := 0; step <= len(forest.nodes); step++ {
		node, exists := forest.nodes[treeNodeKey{tree: treeID, node: nodeID}]
		if !exists {
			return 0, fmt.Errorf("tree %d references missing node %d", treeID, nodeID)
		}
		if node.mode == "LEAF" {
			return nodeID, nil
		}
		if node.feature < 0 || node.feature >= int64(len(row)) {
			return 0, fmt.Errorf("tree %d node %d references feature %d for width %d", treeID, nodeID, node.feature, len(row))
		}
		value := row[node.feature]
		goTrue := false
		if math.IsNaN(float64(value)) {
			goTrue = node.missingTrue
		} else {
			switch node.mode {
			case "BRANCH_LEQ":
				goTrue = value <= node.value
			case "BRANCH_LT":
				goTrue = value < node.value
			case "BRANCH_GTE":
				goTrue = value >= node.value
			case "BRANCH_GT":
				goTrue = value > node.value
			case "BRANCH_EQ":
				goTrue = value == node.value
			case "BRANCH_NEQ":
				goTrue = value != node.value
			default:
				return 0, fmt.Errorf("tree %d node %d has unsupported mode %q", treeID, nodeID, node.mode)
			}
		}
		if goTrue {
			nodeID = node.trueNode
		} else {
			nodeID = node.falseNode
		}
	}
	return 0, fmt.Errorf("tree %d contains a cycle", treeID)
}

func applyMLPostTransform(input *Tensor, transform string) (*Tensor, error) {
	switch transform {
	case "", "NONE":
		return input, nil
	case "LOGISTIC":
		return unary("post-transform logistic", input, sigmoidScalar)
	default:
		return nil, fmt.Errorf("unsupported post_transform %q", transform)
	}
}

func classifierPostTransform(attributes map[string]protoAttribute) (string, error) {
	transform := mlAttrString(attributes, "post_transform", "NONE")
	switch transform {
	case "", "NONE", "LOGISTIC":
		return transform, nil
	default:
		return "", fmt.Errorf("unsupported classifier post_transform %q", transform)
	}
}

func mlClassLabels(attributes map[string]protoAttribute) ([]any, DType, error) {
	if value, present := attributes["classlabels_ints"]; present {
		labels := make([]any, len(value.ints))
		for index, label := range value.ints {
			labels[index] = label
		}
		return labels, DTypeInt64, nil
	}
	if value, present := attributes["classlabels_int64s"]; present {
		labels := make([]any, len(value.ints))
		for index, label := range value.ints {
			labels[index] = label
		}
		return labels, DTypeInt64, nil
	}
	if value, present := attributes["classlabels_strings"]; present {
		labels := make([]any, len(value.strings))
		for index, label := range value.strings {
			labels[index] = string(label)
		}
		return labels, DTypeString, nil
	}
	if value, present := attributes["classlabels_floats"]; present {
		labels := make([]any, len(value.floats))
		for index, label := range value.floats {
			labels[index] = label
		}
		return labels, DTypeFloat32, nil
	}
	return nil, DTypeUnknown, fmt.Errorf("classifier has no supported class labels")
}

func newClassLabelTensor(dtype DType, labels []any, rows int) (*Tensor, error) {
	switch dtype {
	case DTypeInt64:
		values := make([]int64, rows)
		return newInt64Tensor([]int{rows}, values)
	case DTypeString:
		values := make([]string, rows)
		return newStringTensor([]int{rows}, values)
	case DTypeFloat32:
		values := make([]float32, rows)
		return newFloat32Tensor([]int{rows}, values)
	default:
		return nil, fmt.Errorf("unsupported class label dtype %s", dtypeName(dtype))
	}
}

func setClassLabel(destination *Tensor, row int, labels []any, dtype DType, index int) {
	switch dtype {
	case DTypeInt64:
		destination.int64Data[row] = labels[index].(int64)
	case DTypeString:
		destination.stringData[row] = labels[index].(string)
	case DTypeFloat32:
		destination.data[row] = labels[index].(float32)
	}
}

func softmaxRow(values []float32) {
	if len(values) == 0 {
		return
	}
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	var sum float64
	for index, value := range values {
		values[index] = float32(math.Exp(float64(value - maximum)))
		sum += float64(values[index])
	}
	for index := range values {
		values[index] = float32(float64(values[index]) / sum)
	}
}

func allZero(values []int64) bool {
	for _, value := range values {
		if value != 0 {
			return false
		}
	}
	return true
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func mlAttrFloats(attributes map[string]protoAttribute, name string, required bool) ([]float32, error) {
	value, present := attributes[name]
	if !present {
		if required {
			return nil, fmt.Errorf("missing attribute %s", name)
		}
		return nil, nil
	}
	return append([]float32(nil), value.floats...), nil
}

func mlAttrInts(attributes map[string]protoAttribute, name string, required bool) ([]int64, error) {
	value, present := attributes[name]
	if !present {
		if required {
			return nil, fmt.Errorf("missing attribute %s", name)
		}
		return nil, nil
	}
	return append([]int64(nil), value.ints...), nil
}

func mlAttrStrings(attributes map[string]protoAttribute, name string, required bool) ([]string, error) {
	value, present := attributes[name]
	if !present {
		if required {
			return nil, fmt.Errorf("missing attribute %s", name)
		}
		return nil, nil
	}
	result := make([]string, len(value.strings))
	for index, item := range value.strings {
		result[index] = string(item)
	}
	return result, nil
}

func mlAttrString(attributes map[string]protoAttribute, name, fallback string) string {
	value, present := attributes[name]
	if !present || len(value.string) == 0 {
		return fallback
	}
	return string(value.string)
}
