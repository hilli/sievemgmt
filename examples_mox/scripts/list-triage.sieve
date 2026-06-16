require ["fileinto", "mime"];

if anyof (
  header :contains "list-id" "example.localhost",
  address :contains "from" "newsletter"
) {
  fileinto "Lists";
  stop;
}

keep;
