package plot

type Position string

const (
	PositionTop    Position = "top"
	PositionBottom Position = "bottom"
	PositionLeft   Position = "left"
	PositionRight  Position = "right"
)

type LabelPosition string

const (
	LabelPositionTop               LabelPosition = "top"
	LabelPositionBottom            LabelPosition = "bottom"
	LabelPositionLeft              LabelPosition = "left"
	LabelPositionRight             LabelPosition = "right"
	LabelPositionInside            LabelPosition = "inside"
	LabelPositionInsideLeft        LabelPosition = "insideLeft"
	LabelPositionInsideRight       LabelPosition = "insideRight"
	LabelPositionInsideTop         LabelPosition = "insideTop"
	LabelPositionInsideBottom      LabelPosition = "insideBottom"
	LabelPositionInsideTopLeft     LabelPosition = "insideTopLeft"
	LabelPositionInsideBottomLeft  LabelPosition = "insideBottomLeft"
	LabelPositionInsideTopRight    LabelPosition = "insideTopRight"
	LabelPositionInsideBottomRight LabelPosition = "insideBottomRight"
)
