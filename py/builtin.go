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

func builtInFunc(ipcAddress string, executionID string) string {
	return fmt.Sprintf(`
import socket
import struct
import os
import json

class insyra:
	# Unique execution ID for this run
	execution_id = "%s"
	ipc_address = r"%s"

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

	@staticmethod
	def _connect_ipc():
		addr = insyra.ipc_address
		# Try AF_UNIX if available (Linux/macOS/Windows+Py3.9)
		if hasattr(socket, 'AF_UNIX'):
			try:
				s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
				s.connect(addr)
				return s.makefile('rwb', buffering=0)
			except Exception:
				pass
		
		# Fallback for Windows Named Pipe using standard file IO
		if os.name == 'nt':
			try:
				return open(addr, 'r+b', buffering=0)
			except Exception:
				pass
		
		raise Exception("Failed to connect to IPC address: " + addr)

	@staticmethod
	def _write_msg(f, b):
		f.write(struct.pack('<I', len(b)))
		f.write(b)
		f.flush()

	@staticmethod
	def _read_msg(f):
		header = f.read(4)
		if len(header) < 4: return None
		l = struct.unpack('<I', header)[0]
		data = f.read(l)
		while len(data) < l:
			more = f.read(l - len(data))
			if not more: break
			data += more
		return data

	# Function to return result or error back to Go side
	def Return(result=None, error=None):
		global sent
		sys.stdout.flush()
		sys.stderr.flush()
		payload = {"execution_id": insyra.execution_id, "data": [insyra._normalize_result(result), error]}
		
		try:
			f = insyra._connect_ipc()
			insyra._write_msg(f, json.dumps(payload).encode('utf-8'))
			# Wait for ack
			insyra._read_msg(f)
			f.close()
		except Exception as e:
			sys.stderr.write(f"IPC failed: {e}\n")
			sys.stderr.flush()
			raise e

		sent = True
		import os
		os._exit(0)

# alias of insyra.Return for convenience
insyra_return = insyra.Return
`, executionID, ipcAddress,
		pyReturnTypeKey,
		pyReturnDataKey,
		pyReturnColumnsKey,
		pyReturnIndexKey,
		pyReturnNameKey,
		pyReturnTypeDataTable,
		pyReturnTypeDataFrame,
		pyReturnTypeDataList,
		pyReturnTypeSeries,
	)
}
