require ["fileinto", "mime"];

if header :contains "subject" "[sievemgmt-example]" {
  fileinto "Examples";
  stop;
}

if anyof (
  header :contains "list-id" "example.localhost",
  address :contains "from" "newsletter"
) {
  fileinto "Lists";
  stop;
}

keep;
