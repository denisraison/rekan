generator go {
  output_type "go"
  output_dir ".."
  client_package_name "github.com/denisraison/rekan/api/internal/baml"
  version "0.219.0"
  on_generate "gofmt -w . && goimports -w . && go mod tidy"
}
