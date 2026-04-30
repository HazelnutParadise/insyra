## Dumps rnorm-generated clustering/KNN data so Go tests can use byte-identical
## inputs (without re-implementing R's RNG).

set.seed(5001)
hc10 <- matrix(round(rnorm(30), 4), nrow = 10)
for (i in seq_len(nrow(hc10))) {
  cat(sprintf("hc10_row%d:", i - 1),
      paste(hc10[i, ], collapse = ","), "\n", sep = "")
}

set.seed(5002)
ntrain <- 20
knn_train2 <- matrix(round(rnorm(ntrain * 3), 3), nrow = ntrain)
for (i in seq_len(nrow(knn_train2))) {
  cat(sprintf("knn3c_train_row%d:", i - 1),
      paste(knn_train2[i, ], collapse = ","), "\n", sep = "")
}
