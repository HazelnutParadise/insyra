## Dumps rnorm-generated KMeans data so the Go test feeds byte-identical input.

set.seed(6001)
n <- 10
km2 <- rbind(
  cbind(rnorm(n, mean = 0,  sd = 0.5), rnorm(n, mean = 0,  sd = 0.5)),
  cbind(rnorm(n, mean = 10, sd = 0.5), rnorm(n, mean = 10, sd = 0.5)))
km2 <- round(km2, 4)
for (i in seq_len(nrow(km2))) {
  cat(sprintf("km2blob_row%d:", i - 1),
      paste(km2[i, ], collapse = ","), "\n", sep = "")
}

set.seed(6002)
m <- 10
km4 <- rbind(
  cbind(rnorm(m, 0, 0.4), rnorm(m, 0, 0.4)),
  cbind(rnorm(m, 5, 0.4), rnorm(m, 0, 0.4)),
  cbind(rnorm(m, 0, 0.4), rnorm(m, 5, 0.4)),
  cbind(rnorm(m, 5, 0.4), rnorm(m, 5, 0.4)))
km4 <- round(km4, 4)
for (i in seq_len(nrow(km4))) {
  cat(sprintf("km4blob_row%d:", i - 1),
      paste(km4[i, ], collapse = ","), "\n", sep = "")
}
