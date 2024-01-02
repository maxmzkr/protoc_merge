# Test
```
rm -rf example/merged/*; go build -o protoc-gen-merge ./main.go && protoc --experimental_allow_proto3_optional --plugin=./protoc-gen-merge --merge_out=./example/merged --merge_opt=prefix=example/base,prefix=example/merge,prefix=example/merged,package=example.base,package=example.merge,package=example.merged,paths=example.base.Test ./example/base/test.proto ./example/merge/test.proto && (find ./example/merged -name '*.proto' | xargs cat)
```
