package py

import "fmt"

func builtInFunc(port string, executionID string) string {
	return fmt.Sprintf(`
class insyra:
	# Unique execution ID for this run
	execution_id = "%s"

	# Function to return result or error back to Go side
	def Return(result=None, error=None, url="http://localhost:%v/pyresult"):
		global sent
		sys.stdout.flush()
		sys.stderr.flush()
		payload = {"execution_id": insyra.execution_id, "data": [result, error]}
		response = requests.post(url, json=payload)
		if response.status_code != 200:
			raise Exception(f"Failed to send result: {response.status_code}")
		sent = True
		import os
		os._exit(0)

# alias of insyra.Return for convenience
insyra_return = insyra.Return
`, executionID, port)
}
