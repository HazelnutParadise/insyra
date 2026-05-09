## Dumps rnorm-generated data so the Go test file uses byte-identical inputs.
set.seed(42); cat("zsA_largeN_two:", paste(round(rnorm(200, 100, 10), 4), collapse=","), "\n", sep="")
set.seed(43); cat("zsA_largeN_grt:", paste(round(rnorm(200, 100, 10), 4), collapse=","), "\n", sep="")
set.seed(44); cat("zsA_largeN_less:", paste(round(rnorm(200, 100, 10), 4), collapse=","), "\n", sep="")
set.seed(7); cat("ztA_largeN_d1:", paste(round(rnorm(200, 100, 5), 3), collapse=","), "\n", sep="")
set.seed(8); cat("ztA_largeN_d2:", paste(round(rnorm(250, 102, 5), 3), collapse=","), "\n", sep="")
