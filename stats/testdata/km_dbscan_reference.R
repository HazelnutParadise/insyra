## Reference values for KMeans (base R kmeans) and DBSCAN (dbscan::dbscan).
##
## KMeans
##   Random initialisation differs between R's RNG and insyra's LCG, so we
##   compare statistics that are invariant under cluster relabeling and that
##   converge to the same global optimum given enough restarts:
##     · TotSS, TotWithinSS, BetweenSS (all unaffected by relabeling)
##     · sort(WithinSS) ascending
##     · sort(Size) ascending
##   With well-separated data and nstart >= 25 in both implementations, the
##   global minimum is reliably hit. Cluster assignments are compared
##   pairwise via the adjusted Rand index in tests; here we just emit the
##   invariant statistics.
##
## DBSCAN
##   Deterministic given (eps, minPts) and a fixed point visit order. Both
##   R dbscan::dbscan and insyra walk points in their input order, so cluster
##   *numbering* matches up to a canonical relabeling: rename each cluster
##   by the smallest 1-indexed point that belongs to it. We emit:
##     · canonicalised cluster vector (0 = noise)
##     · is_corepoint vector
##     · cluster count, noise count

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")
emit_int <- function(key, val) cat(key, "=", as.integer(val), "\n", sep = "")
emit_bool <- function(key, val) cat(key, "=", if (val) "true" else "false", "\n", sep = "")

# Canonical relabeling: cluster IDs reassigned so that cluster 1 is whichever
# cluster the smallest-index non-noise point belongs to, etc. Noise stays 0.
canonicalize <- function(cluster) {
  out <- integer(length(cluster))
  next_id <- 0L
  mapping <- list()                # name -> integer
  for (i in seq_along(cluster)) {
    c <- cluster[i]
    if (c == 0) { out[i] <- 0L; next }
    key <- as.character(c)
    if (is.null(mapping[[key]])) {
      next_id <- next_id + 1L
      mapping[[key]] <- next_id
    }
    out[i] <- mapping[[key]]
  }
  out
}

emit_kmeans <- function(prefix, data, k, nstart = 50) {
  set.seed(0)   # for R's own restarts; insyra has its own seed-controlled init
  fit <- kmeans(data, centers = k, nstart = nstart)
  emit(paste0(prefix, ".totSS"),       fit$totss)
  emit(paste0(prefix, ".totWithinSS"), fit$tot.withinss)
  emit(paste0(prefix, ".betweenSS"),   fit$betweenss)
  ws <- sort(fit$withinss)
  for (i in seq_along(ws)) emit(paste0(prefix, ".withinSS_sorted[", i - 1, "]"), ws[i])
  sz <- sort(fit$size)
  for (i in seq_along(sz)) emit_int(paste0(prefix, ".size_sorted[", i - 1, "]"), sz[i])
}

emit_dbscan <- function(prefix, data, eps, minPts) {
  fit <- dbscan::dbscan(data, eps = eps, minPts = minPts)
  cl  <- canonicalize(fit$cluster)
  cp  <- if (!is.null(attr(fit, "is.corepoint"))) attr(fit, "is.corepoint")
         else fit$is.corepoint
  if (is.null(cp)) {
    cp <- dbscan::is.corepoint(data, eps = eps, minPts = minPts)
  }
  for (i in seq_along(cl)) {
    emit_int(paste0(prefix, ".cluster[", i - 1, "]"), cl[i])
    emit_bool(paste0(prefix, ".core[",   i - 1, "]"), cp[i])
  }
  emit_int(paste0(prefix, ".n_clusters"), max(cl))
  emit_int(paste0(prefix, ".n_noise"),    sum(cl == 0))
}

# ============================================================
# KMeans: well-separated synthetic data
# ============================================================
# Three obvious blobs in 2D, n=15 (5 per blob). Global optimum is unique.
km3 <- rbind(
  c( 0,  0), c( 0.2,  0.1), c(-0.1,  0.3), c( 0.1, -0.2), c(-0.2, -0.1),
  c(10, 10), c(10.1,  9.8), c( 9.9, 10.2), c(10.2, 10.3), c( 9.8,  9.9),
  c(20,  0), c(20.3,  0.1), c(19.7, -0.1), c(20.1,  0.2), c(19.9, -0.2))
emit_kmeans("km3blob", km3, 3)

# Two blobs, larger, n=20
set.seed(6001)
n <- 10
km2 <- rbind(
  cbind(rnorm(n, mean = 0,  sd = 0.5), rnorm(n, mean = 0,  sd = 0.5)),
  cbind(rnorm(n, mean = 10, sd = 0.5), rnorm(n, mean = 10, sd = 0.5)))
km2 <- round(km2, 4)
emit_kmeans("km2blob_n20", km2, 2)

# 4 blobs, n=40 (10 each)
set.seed(6002)
m <- 10
km4 <- rbind(
  cbind(rnorm(m, 0, 0.4),  rnorm(m, 0, 0.4)),
  cbind(rnorm(m, 5, 0.4),  rnorm(m, 0, 0.4)),
  cbind(rnorm(m, 0, 0.4),  rnorm(m, 5, 0.4)),
  cbind(rnorm(m, 5, 0.4),  rnorm(m, 5, 0.4)))
km4 <- round(km4, 4)
emit_kmeans("km4blob_n40", km4, 4)

# ============================================================
# DBSCAN: distinct clusters with isolated noise
# ============================================================
# 3 close points + 1 distant noise (matches existing TestDBSCANDetectsNoiseAndCorePoints)
db_basic <- rbind(c(0, 0), c(0.1, 0), c(0, 0.1), c(8, 8))
emit_dbscan("db_basic", db_basic, eps = 0.25, minPts = 3)

# Two clusters + noise
db_2cluster <- rbind(
  c(0, 0), c(0.1, 0.05), c(-0.05, 0.1), c(0.05, -0.05), c(0.1, 0.1),
  c(10, 10), c(10.1, 10), c(10, 10.1), c(10.1, 10.1), c(10.05, 9.95),
  c(50, 50))   # isolated noise
emit_dbscan("db_2cluster", db_2cluster, eps = 0.3, minPts = 3)

# 3 clusters of varying density
db_3cluster <- rbind(
  c(0, 0), c(0.05, 0.05), c(-0.05, 0), c(0, 0.05), c(0.02, -0.02),
  c(5, 5), c(5.1, 5), c(5, 5.1), c(5.05, 5.05),
  c(10, 0), c(10, 0.1), c(10.1, 0), c(10.05, 0.05),
  c(20, 20))   # noise
emit_dbscan("db_3cluster", db_3cluster, eps = 0.2, minPts = 3)
