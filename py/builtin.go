package py

import "fmt"

const (
	pyReturnTypeKey    = "_insyra_type"
	pyReturnDataKey    = "data"
	pyReturnColumnsKey = "columns"
	pyReturnIndexKey   = "index"
	pyReturnNameKey    = "name"

	pyReturnTypeDataTable = "datatable"
	pyReturnTypeDataFrame = "dataframe"
	pyReturnTypeDataList  = "datalist"
	pyReturnTypeSeries    = "series"
)

func builtInFunc(port string, executionID string) string {
	return fmt.Sprintf(`
class insyra:
	# Unique execution ID for this run
	execution_id = "%s"

	_TYPE_KEY = "%s"
	_DATA_KEY = "%s"
	_COLUMNS_KEY = "%s"
	_INDEX_KEY = "%s"
	_NAME_KEY = "%s"
	_TYPE_DATATABLE = "%s"
	_TYPE_DATAFRAME = "%s"
	_TYPE_DATALIST = "%s"
	_TYPE_SERIES = "%s"

	@staticmethod
	def _normalize_result(result):
		if result is None:
			return None

		if isinstance(result, pd.DataFrame):
			payload = {
				insyra._TYPE_KEY: insyra._TYPE_DATATABLE,
				insyra._DATA_KEY: result.to_numpy().tolist(),
				insyra._COLUMNS_KEY: [str(c) for c in result.columns],
				insyra._INDEX_KEY: [str(i) for i in result.index],
			}
			name_value = getattr(result, "name", None)
			if name_value is not None:
				payload[insyra._NAME_KEY] = str(name_value)
			return payload

		if isinstance(result, pd.Series):
			payload = {
				insyra._TYPE_KEY: insyra._TYPE_DATALIST,
				insyra._DATA_KEY: result.tolist(),
			}
			name_value = result.name
			if name_value is not None:
				payload[insyra._NAME_KEY] = str(name_value)
			return payload

		if isinstance(result, pl.DataFrame):
			payload = {
				insyra._TYPE_KEY: insyra._TYPE_DATATABLE,
				insyra._DATA_KEY: result.to_numpy().tolist(),
				insyra._COLUMNS_KEY: [str(c) for c in result.columns],
			}
			name_value = getattr(result, "name", None)
			if name_value is not None:
				payload[insyra._NAME_KEY] = str(name_value)
			return payload

		if isinstance(result, pl.Series):
			payload = {
				insyra._TYPE_KEY: insyra._TYPE_DATALIST,
				insyra._DATA_KEY: result.to_list(),
			}
			name_value = result.name
			if name_value is not None:
				payload[insyra._NAME_KEY] = str(name_value)
			return payload

		return result

	# Function to return result or error back to Go side
	def Return(result=None, error=None, url="http://127.0.0.1:%v/pyresult"):
		global sent
		sys.stdout.flush()
		sys.stderr.flush()
		payload = {"execution_id": insyra.execution_id, "data": [insyra._normalize_result(result), error]}
		response = requests.post(url, json=payload)
		if response.status_code != 200:
			raise Exception(f"Failed to send result: {response.status_code}")
		sent = True
		import os
		os._exit(0)

# alias of insyra.Return for convenience
insyra_return = insyra.Return
`, executionID,
		pyReturnTypeKey,
		pyReturnDataKey,
		pyReturnColumnsKey,
		pyReturnIndexKey,
		pyReturnNameKey,
		pyReturnTypeDataTable,
		pyReturnTypeDataFrame,
		pyReturnTypeDataList,
		pyReturnTypeSeries,
		port,
	)
}
