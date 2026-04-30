## Reference values for stats/clustering.go and stats/knn.go using base R only.
##
## Sources of truth:
##   hierarchical : hclust(dist(data), method=...) — merge, height, order.
##                  Insyra mirrors R's output convention exactly: negative
##                  integers in $merge denote leaves (1-indexed), positive
##                  integers denote previous merge steps; $order is 1-indexed.
##   silhouette   : hand-computed using base R (avoids the cluster package
##                  dependency). Matches the standard formulas:
##                    a(i) = mean distance from i to other members of i's cluster
##                    b(i) = min over other clusters of mean distance from i to that cluster
##                    s(i) = (b(i) - a(i)) / max(a(i), b(i))
##                    s(singleton) = 0
##   knn classify : hand-computed using sort() on Euclidean distances.
##   knn regress  : hand-computed; uniform mean of nearest-k targets.
##   knn neighbors: indices and distances.
##
## All test datasets are chosen with distinct distances so tiebreaker
## conventions don't affect the comparison.

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")
emit_int <- function(key, val) cat(key, "=", as.integer(val), "\n", sep = "")

emit_hclust <- function(prefix, mat, method) {
  d <- dist(mat)
  h <- hclust(d, method = method)
  n <- nrow(mat)
  for (i in seq_len(n - 1)) {
    emit_int(paste0(prefix, ".merge[", i - 1, "].a"), h$merge[i, 1])
    emit_int(paste0(prefix, ".merge[", i - 1, "].b"), h$merge[i, 2])
    emit(paste0(prefix, ".height[", i - 1, "]"),     h$height[i])
  }
  for (i in seq_len(n)) {
    emit_int(paste0(prefix, ".order[", i - 1, "]"), h$order[i])
  }
}

# ---- silhouette (hand-computed) ----
emit_silhouette <- function(prefix, mat, labels) {
  n <- nrow(mat)
  D <- as.matrix(dist(mat))
  cls <- sort(unique(labels))
  s_vals <- numeric(n)
  for (i in seq_len(n)) {
    own <- labels[i]
    own_idx <- setdiff(which(labels == own), i)
    if (length(own_idx) == 0) {
      s_vals[i] <- 0
      next
    }
    a_i <- mean(D[i, own_idx])
    other_clusters <- cls[cls != own]
    b_candidates <- sapply(other_clusters, function(c)
      mean(D[i, which(labels == c)]))
    b_i <- min(b_candidates)
    s_vals[i] <- (b_i - a_i) / max(a_i, b_i)
    emit(paste0(prefix, ".a[", i - 1, "]"), a_i)
    emit(paste0(prefix, ".b[", i - 1, "]"), b_i)
    emit(paste0(prefix, ".s[", i - 1, "]"), s_vals[i])
    next
  }
  # also emit singleton-zero entries
  for (i in seq_len(n)) {
    own <- labels[i]
    own_idx <- setdiff(which(labels == own), i)
    if (length(own_idx) == 0) {
      emit(paste0(prefix, ".a[", i - 1, "]"), 0)
      emit(paste0(prefix, ".b[", i - 1, "]"), 0)
      emit(paste0(prefix, ".s[", i - 1, "]"), 0)
    }
  }
  emit(paste0(prefix, ".avg"), mean(s_vals))
}

# ---- KNN classify ----
# Class column order is first-appearance (matches insyra's orderedClasses, not
# alphabetical sort). Tie-breaking for the prediction is insyra-specific (it
# picks the class with smaller mean distance among the k nearest); we emit
# probabilities only and compare predictions only on cases without ties.
emit_knn_classify <- function(prefix, train, test, train_labels, k) {
  classes <- unique(train_labels)   # preserves first-appearance order
  for (q in seq_len(nrow(test))) {
    dists <- apply(train, 1, function(r) sqrt(sum((r - test[q, ])^2)))
    o <- order(dists)[seq_len(k)]
    nearest <- train_labels[o]
    counts <- sapply(classes, function(c) sum(nearest == c))
    for (ci in seq_along(classes)) {
      emit(paste0(prefix, ".prob[", q - 1, "][", ci - 1, "]"),
           counts[ci] / k)
    }
  }
  for (ci in seq_along(classes)) {
    emit(paste0(prefix, ".class[", ci - 1, "]_str"),
         paste0("\"", classes[ci], "\""))
  }
}

# ---- KNN regress (uniform) ----
emit_knn_regress <- function(prefix, train, test, train_targets, k) {
  for (q in seq_len(nrow(test))) {
    dists <- apply(train, 1, function(r) sqrt(sum((r - test[q, ])^2)))
    o <- order(dists)[seq_len(k)]
    emit(paste0(prefix, ".pred[", q - 1, "]"), mean(train_targets[o]))
  }
}

# ---- KNN neighbors (indices + distances) ----
# insyra reports 1-indexed neighbor indices (see stats/knn.go:128 — adds 1 to
# the internal 0-indexed result before returning), matching R's own
# convention. We emit 1-indexed here so reference values plug in directly.
emit_knn_neighbors <- function(prefix, train, test, k) {
  for (q in seq_len(nrow(test))) {
    dists <- apply(train, 1, function(r) sqrt(sum((r - test[q, ])^2)))
    o <- order(dists)[seq_len(k)]
    for (i in seq_len(k)) {
      emit_int(paste0(prefix, ".idx[", q - 1, "][", i - 1, "]"), o[i])
      emit(paste0(prefix, ".dist[", q - 1, "][", i - 1, "]"),    dists[o[i]])
    }
  }
}

# ============================================================
# Hierarchical clustering — distinct distances so no tie-breaking ambiguity
# ============================================================
hc_data <- rbind(
  c(0.0, 0.0),
  c(0.5, 1.1),
  c(8.3, 7.9),
  c(9.2, 8.4),
  c(15.1, 0.7))

emit_hclust("hc_complete", hc_data, "complete")
emit_hclust("hc_single",   hc_data, "single")
emit_hclust("hc_average",  hc_data, "average")
emit_hclust("hc_ward.D",   hc_data, "ward.D")
emit_hclust("hc_ward.D2",  hc_data, "ward.D2")

# Larger n=10, 3-D
set.seed(5001)
hc10 <- matrix(round(rnorm(30), 4), nrow = 10)
emit_hclust("hc10_complete", hc10, "complete")
emit_hclust("hc10_average",  hc10, "average")

# ============================================================
# Silhouette
# ============================================================
sil_data <- rbind(
  c(0, 0), c(0.2, 0.1), c(0.1, 0.3),
  c(10, 10), c(10.1, 9.9), c(9.9, 10.1))
sil_labels <- c(1, 1, 1, 2, 2, 2)
emit_silhouette("sil_clean", sil_data, sil_labels)

# Mixed clusters with variable density
sil_data2 <- rbind(
  c(0, 0), c(0.5, 0), c(0, 0.5),
  c(5, 5), c(5.3, 4.8), c(4.7, 5.2), c(5.1, 5.4),
  c(20, 20))
sil_labels2 <- c(1, 1, 1, 2, 2, 2, 2, 3)
emit_silhouette("sil_mixed", sil_data2, sil_labels2)

# ============================================================
# KNN
# ============================================================
knn_train <- rbind(
  c(0, 0), c(0, 1), c(1, 0),
  c(10, 10), c(10, 11), c(11, 10))
knn_test <- rbind(c(0.1, 0.2), c(10.2, 10.1), c(5, 5))
knn_labels <- c("red", "red", "red", "blue", "blue", "blue")

emit_knn_classify("knn_basic_k3", knn_train, knn_test, knn_labels, 3)

# Larger train set, n_train=20, n_test=5, three classes
set.seed(5002)
ntrain <- 20
knn_train2 <- matrix(round(rnorm(ntrain * 3), 3), nrow = ntrain)
knn_labels2 <- c(rep("a", 7), rep("b", 7), rep("c", 6))
knn_test2 <- knn_train2[c(1, 5, 10, 15, 20), ]
emit_knn_classify("knn_3class_k5", knn_train2, knn_test2, knn_labels2, 5)

# Regression
reg_train <- rbind(c(0, 0), c(0, 1), c(10, 10), c(10, 11))
reg_test <- rbind(c(0.1, 0.2), c(9.9, 10.1))
reg_targets <- c(1, 1.5, 9, 9.5)
emit_knn_regress("knn_reg_k2_uniform", reg_train, reg_test, reg_targets, 2)

# Neighbors lookup (indices + distances)
nbr_train <- rbind(
  c(0, 0), c(0, 1), c(1, 0),
  c(10, 10), c(10, 11), c(11, 10))
nbr_test <- rbind(c(0.1, 0.2), c(10.1, 10.2))
emit_knn_neighbors("knn_nbr_k2", nbr_train, nbr_test, 2)
