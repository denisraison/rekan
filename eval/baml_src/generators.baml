generator go {
  output_type "go"
  output_dir "../baml_client"
  client_package_name "github.com/denisraison/rekan/eval/baml_client"
  version "0.219.0"
  on_generate "gofmt -w . && goimports -w . && go mod tidy"
}
