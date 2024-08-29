package insyra

import (
	"github.com/yourusername/insyra/data"
	"github.com/yourusername/insyra/stats"
	"github.com/yourusername/insyra/visualization"
)

// AnalyzeAndVisualize 是一个示例函数，结合了数据加载、统计分析和可视化功能
func AnalyzeAndVisualize(filepath string) error {
	data, err := data.LoadCSV(filepath)
	if err != nil {
		return err
	}

	floatData, err := data.ConvertToFloat(data)
	if err != nil {
		return err
	}

	statsResult := stats.BasicStats(floatData)
	err = visualization.PlotData(statsResult)
	if err != nil {
		return err
	}

	return nil
}
