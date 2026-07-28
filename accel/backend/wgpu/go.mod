// Module github.com/HazelnutParadise/insyra/accel/backend/wgpu is deliberately
// separate from the core insyra module: a subpackage would put six GPU modules
// into every `go get github.com/HazelnutParadise/insyra`, and a build tag would
// not prevent that because the requirement lands in go.mod either way.
module github.com/HazelnutParadise/insyra/accel/backend/wgpu

go 1.25.12

require (
	github.com/HazelnutParadise/insyra v0.3.0
	github.com/gogpu/gputypes v0.5.1
	github.com/gogpu/wgpu v0.30.23
)

require (
	github.com/HazelnutParadise/Go-Utils v0.8.2 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/go-webgpu/goffi v0.6.2 // indirect
	github.com/go-webgpu/webgpu v0.5.4 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gogpu/gpucontext v0.21.1 // indirect
	github.com/gogpu/naga v0.17.16 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/petermattis/goid v0.0.0-20260701081913-4f67fd55d3b4 // indirect
	github.com/richardlehane/mscfb v1.0.7 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/excelize/v2 v2.11.0 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	gorm.io/gorm v1.31.2 // indirect
)

// Local development against the in-repo insyra. Consumers ignore replace
// directives in a dependency's go.mod, so this only affects work in this repo.
replace github.com/HazelnutParadise/insyra => ../../../
