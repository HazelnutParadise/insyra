suppressMessages(library(jsonlite))

ci_by_alt <- function(center, margin, alt) {
  if (alt == "greater") return(c(center - margin, Inf))
  if (alt == "less") return(c(-Inf, center + margin))
  c(center - margin, center + margin)
}

fisher_ci <- function(r, n, cl = 0.95) {
  z <- 0.5 * log((1 + r) / (1 - r))
  se <- 1 / sqrt(n - 3)
  zcrit <- qnorm(1 - (1 - cl) / 2)
  zl <- z - zcrit * se
  zu <- z + zcrit * se
  c(tanh(zl), tanh(zu))
}

correlation_inference <- function(corr, n) {
  t <- corr * sqrt(n - 2) / sqrt(1 - corr * corr)
  p <- 2 * (1 - pt(abs(t), n - 2))
  ci <- fisher_ci(corr, n, 0.95)
  list(t = t, p = p, df = as.double(n - 2), ci = ci)
}

one_way_stats <- function(groups) {
  all_vals <- unlist(groups)
  labels <- unlist(mapply(function(i, g) rep(i, length(g)), seq_along(groups), groups))
  total_mean <- mean(all_vals)
  ssb <- 0
  ssw <- 0
  for (i in seq_along(groups)) {
    vals <- all_vals[labels == i]
    mean_i <- mean(vals)
    ssb <- ssb + length(vals) * (mean_i - total_mean)^2
    ssw <- ssw + sum((vals - mean_i)^2)
  }
  dfb <- length(groups) - 1
  dfw <- length(all_vals) - length(groups)
  f <- (ssb / dfb) / (ssw / dfw)
  p <- 1 - pf(f, dfb, dfw)
  eta <- ssb / (ssb + ssw)
  list(ssb = ssb, ssw = ssw, dfb = as.double(dfb), dfw = as.double(dfw), f = f, p = p, eta = eta)
}

two_way_stats <- function(a_levels, b_levels, cells) {
  all_values <- c()
  fa <- c()
  fb <- c()
  counts <- c()

  for (i in 0:(a_levels - 1)) {
    for (j in 0:(b_levels - 1)) {
      idx <- i * b_levels + j + 1
      cell <- as.double(unlist(cells[[idx]]))
      counts <- c(counts, length(cell))
      all_values <- c(all_values, cell)
      fa <- c(fa, rep(i, length(cell)))
      fb <- c(fb, rep(j, length(cell)))
    }
  }

  total_mean <- mean(all_values)

  ssa <- 0
  for (i in 0:(a_levels - 1)) {
    vals <- all_values[fa == i]
    ssa <- ssa + length(vals) * (mean(vals) - total_mean)^2
  }

  ssb <- 0
  for (j in 0:(b_levels - 1)) {
    vals <- all_values[fb == j]
    ssb <- ssb + length(vals) * (mean(vals) - total_mean)^2
  }

  cell_means <- c()
  for (i in 0:(a_levels - 1)) {
    for (j in 0:(b_levels - 1)) {
      vals <- all_values[fa == i & fb == j]
      cell_means <- c(cell_means, mean(vals))
    }
  }

  ssab <- 0
  for (i in 0:(a_levels - 1)) {
    mean_a <- mean(all_values[fa == i])
    for (j in 0:(b_levels - 1)) {
      mean_b <- mean(all_values[fb == j])
      idx <- i * b_levels + j + 1
      n_ij <- counts[idx]
      mu_ij <- cell_means[idx]
      ssab <- ssab + n_ij * (mu_ij - mean_a - mean_b + total_mean)^2
    }
  }

  ssw <- 0
  for (i in 0:(a_levels - 1)) {
    for (j in 0:(b_levels - 1)) {
      idx <- i * b_levels + j + 1
      vals <- all_values[fa == i & fb == j]
      ssw <- ssw + sum((vals - cell_means[idx])^2)
    }
  }

  dfa <- a_levels - 1
  dfb <- b_levels - 1
  dfab <- dfa * dfb
  dfw <- length(all_values) - a_levels * b_levels

  msa <- ssa / dfa
  msb <- ssb / dfb
  msab <- ssab / dfab
  msw <- ssw / dfw

  fa_stat <- msa / msw
  fb_stat <- msb / msw
  fab_stat <- msab / msw

  pa <- 1 - pf(fa_stat, dfa, dfw)
  pb <- 1 - pf(fb_stat, dfb, dfw)
  pab <- 1 - pf(fab_stat, dfab, dfw)

  list(
    ssa = ssa, ssb = ssb, ssab = ssab, ssw = ssw,
    dfa = as.double(dfa), dfb = as.double(dfb), dfab = as.double(dfab), dfw = as.double(dfw),
    fa = fa_stat, fb = fb_stat, fab = fab_stat,
    pa = pa, pb = pb, pab = pab,
    etaa = ssa / (ssa + ssw),
    etab = ssb / (ssb + ssw),
    etaab = ssab / (ssab + ssw),
    total_ss = ssa + ssb + ssab + ssw
  )
}

rm_stats <- function(subjects) {
  m <- do.call(rbind, lapply(subjects, function(x) as.double(unlist(x))))
  n <- nrow(m)
  k <- ncol(m)
  grand <- mean(m)
  cond_means <- colMeans(m)
  subj_means <- rowMeans(m)

  ss_factor <- n * sum((cond_means - grand)^2)
  ss_subject <- k * sum((subj_means - grand)^2)
  ss_total <- sum((m - grand)^2)
  ss_within <- ss_total - ss_factor - ss_subject

  df_factor <- k - 1
  df_subject <- n - 1
  df_within <- df_factor * df_subject

  ms_factor <- ss_factor / df_factor
  ms_within <- ss_within / df_within
  f <- ms_factor / ms_within
  p <- 1 - pf(f, df_factor, df_within)
  eta <- ss_factor / ss_total

  list(
    ss_factor = ss_factor,
    ss_subject = ss_subject,
    ss_within = ss_within,
    ss_total = ss_total,
    df_factor = as.double(df_factor),
    df_subject = as.double(df_subject),
    df_within = as.double(df_within),
    f = f,
    p = p,
    eta = eta
  )
}

ols_from_matrix <- function(y, X) {
  y <- as.double(y)
  X <- as.matrix(X)
  n <- nrow(X)
  p <- ncol(X) - 1
  df <- n - p - 1

  xtx <- t(X) %*% X
  xty <- t(X) %*% y
  beta <- solve(xtx, xty)
  fit <- as.vector(X %*% beta)
  residuals <- y - fit
  sse <- sum(residuals^2)
  sst <- sum((y - mean(y))^2)
  r2 <- 1 - sse / sst
  adj <- 1 - (1 - r2) * ((n - 1) / df)

  xtx_inv <- solve(xtx)
  mse <- sse / df
  se <- sqrt(diag(mse * xtx_inv))
  tv <- as.vector(beta) / se
  pv <- 2 * (1 - pt(abs(tv), df))
  tcrit <- qt(0.975, df)
  ci <- cbind(as.vector(beta) - tcrit * se, as.vector(beta) + tcrit * se)

  ci_rows <- lapply(seq_len(nrow(ci)), function(i) as.double(ci[i, ]))

  list(
    coefficients = as.vector(beta),
    standard_errors = as.vector(se),
    t_values = as.vector(tv),
    p_values = as.vector(pv),
    confidence_intervals = ci_rows,
    residuals = as.vector(residuals),
    r_squared = r2,
    adj_r_squared = adj
  )
}

pca_stats <- function(rows, n_components = NULL) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  if (is.null(n_components)) {
    n_components <- ncol(m)
  }
  z <- scale(m, center = TRUE, scale = TRUE)
  z[, apply(z, 2, sd) == 0] <- scale(m[, apply(z, 2, sd) == 0, drop = FALSE], center = TRUE, scale = FALSE)
  z[is.na(z)] <- 0
  cv <- cov(z)
  eg <- eigen(cv, symmetric = TRUE)
  vals <- eg$values
  vecs <- eg$vectors
  vals <- vals[1:n_components]
  vecs <- vecs[, 1:n_components, drop = FALSE]
  for (j in seq_len(ncol(vecs))) {
    if (vecs[1, j] < 0) vecs[, j] <- -vecs[, j]
  }
  total_var <- sum(eigen(cv, symmetric = TRUE)$values)
  explained <- vals / total_var * 100
  components <- lapply(seq_len(ncol(vecs)), function(j) as.double(vecs[, j]))
  list(eigenvalues = as.double(vals), explained = as.double(explained), components = components)
}

factor_analysis_stats <- function(rows, extraction, rotation, scoring, nfactors) {
  suppressMessages(library(psych))
  suppressMessages(library(GPArotation))

  rows_to_matrix <- function(rows) {
    nr <- length(rows)
    nc <- max(vapply(rows, length, integer(1)))
    out <- matrix(NA_real_, nrow = nr, ncol = nc)
    for (i in seq_len(nr)) {
      for (j in seq_len(length(rows[[i]]))) {
        v <- rows[[i]][[j]]
        if (!is.null(v)) out[i, j] <- as.double(v)
      }
    }
    out
  }

  m <- rows_to_matrix(rows)
  m <- m[stats::complete.cases(m), , drop = FALSE]
  colnames(m) <- paste0("C", seq_len(ncol(m)))
  rotation <- as.character(rotation)
  scoring <- as.character(scoring)
  score_arg <- switch(scoring,
    "none" = "none",
    "regression" = "regression",
    "bartlett" = "Bartlett",
    "anderson-rubin" = "Anderson",
    "regression"
  )

  if (as.character(extraction) == "pca") {
    fit <- psych::principal(m, nfactors = as.integer(nfactors), rotate = rotation, scores = scoring != "none")
    fit0 <- psych::principal(m, nfactors = as.integer(nfactors), rotate = "none", scores = FALSE)
    initial <- rep(1, ncol(m))
  } else {
    fm <- switch(as.character(extraction), "paf" = "pa", "ml" = "ml", "minres" = "minres", "minres")
    fit <- psych::fa(m, nfactors = as.integer(nfactors), rotate = rotation, fm = fm, scores = score_arg)
    fit0 <- psych::fa(m, nfactors = as.integer(nfactors), rotate = "none", fm = fm, scores = "none")
    initial <- psych::smc(stats::cor(m))
  }

  as_rows <- function(x) {
    if (is.null(x)) return(NULL)
    mm <- as.matrix(x)
    lapply(seq_len(nrow(mm)), function(i) as.double(mm[i, ]))
  }
  pick_v <- function(name) {
    va <- fit$Vaccounted
    if (is.null(va) || is.null(rownames(va)) || !(name %in% rownames(va))) {
      if (name == "Cumulative Var" && !is.null(va) && !is.null(rownames(va)) && ("Proportion Var" %in% rownames(va))) {
        return(cumsum(as.double(va["Proportion Var", seq_len(as.integer(nfactors))])))
      }
      return(rep(NaN, as.integer(nfactors)))
    }
    as.double(va[name, seq_len(as.integer(nfactors))])
  }

  score_cov <- NULL
  if (!is.null(fit$scores)) score_cov <- stats::cov(as.matrix(fit$scores))
  kmo <- psych::KMO(stats::cor(m))
  bart <- psych::cortest.bartlett(stats::cor(m), n = nrow(m))

  list(
    loadings = as_rows(fit$loadings),
    unrotated_loadings = as_rows(fit0$loadings),
    structure = as_rows(if (!is.null(fit$Structure)) fit$Structure else fit$loadings),
    uniquenesses = as.double(fit$uniquenesses),
    communalities = as_rows(cbind(Initial = as.double(initial), Extraction = as.double(fit$communality))),
    phi = as_rows(fit$Phi),
    rotation_matrix = as_rows(fit$rot.mat),
    eigenvalues = as.double(fit$values)[seq_len(as.integer(nfactors))],
    explained_proportion = pick_v("Proportion Var"),
    cumulative_proportion = pick_v("Cumulative Var"),
    sampling_adequacy = as.double(c(kmo$MSAi, kmo$MSA)),
    bartlett = list(
      chi_square = as.double(bart$chisq),
      df = as.double(bart$df),
      p_value = as.double(bart$p.value),
      sample_size = as.double(nrow(m))
    ),
    scores = as_rows(fit$scores),
    score_coefficients = as_rows(fit$weights),
    score_covariance = as_rows(score_cov)
  )
}

lcg_new <- function(seed) {
  env <- new.env(parent = emptyenv())
  env$state <- bitwAnd(as.integer(seed), 0x7fffffff)
  env
}

lcg_next <- function(rng) {
  rng$state <- (1103515245 * rng$state + 12345) %% 2147483647
  rng$state
}

lcg_perm <- function(rng, n) {
  out <- seq_len(n) - 1L
  if (n <= 1) return(out)
  for (i in seq.int(n - 1L, 1L)) {
    j <- (lcg_next(rng) %% (i + 1L)) + 1L
    tmp <- out[i + 1L]
    out[i + 1L] <- out[j]
    out[j] <- tmp
  }
  out
}

nearest_center <- function(row, centers) {
  best_idx <- 1L
  best_dist <- sum((row - centers[[1]])^2)
  if (length(centers) == 1L) return(best_idx)
  for (i in 2:length(centers)) {
    dist <- sum((row - centers[[i]])^2)
    if (dist < best_dist || (abs(dist - best_dist) <= 1e-12 && i < best_idx)) {
      best_idx <- i
      best_dist <- dist
    }
  }
  best_idx
}

farthest_point <- function(data, assignments, centers) {
  best_idx <- 1L
  best_dist <- -1
  for (i in seq_len(nrow(data))) {
    dist <- sum((data[i, ] - centers[[assignments[i]]])^2)
    if (dist > best_dist) {
      best_idx <- i
      best_dist <- dist
    }
  }
  best_idx
}

reorder_kmeans <- function(cluster, centers, size, withinss) {
  keys <- sapply(centers, function(center) paste(sprintf("%.12f", center), collapse = "|"))
  ord <- order(keys, seq_along(keys))
  map <- integer(length(ord))
  for (i in seq_along(ord)) map[ord[i]] <- i
  list(
    cluster = as.integer(map[cluster]),
    centers = lapply(ord, function(i) as.double(centers[[i]])),
    size = as.integer(size[ord]),
    withinss = as.double(withinss[ord])
  )
}

kmeans_single <- function(rows, k, itermax, rng) {
  data <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  n <- nrow(data)
  p <- ncol(data)
  idx <- lcg_perm(rng, n)[seq_len(k)] + 1L
  centers <- lapply(idx, function(i) as.double(data[i, ]))
  assignments <- integer(n)
  iter <- 0L
  changed <- TRUE
  while (iter < itermax && changed) {
    changed <- FALSE
    counts <- integer(k)
    sums <- replicate(k, numeric(p), simplify = FALSE)
    for (i in seq_len(n)) {
      best <- nearest_center(data[i, ], centers)
      if (iter == 0L || assignments[i] != best) changed <- TRUE
      assignments[i] <- best
      counts[best] <- counts[best] + 1L
      sums[[best]] <- sums[[best]] + data[i, ]
    }
    for (c in seq_len(k)) {
      if (counts[c] == 0L) {
        farthest <- farthest_point(data, assignments, centers)
        origin <- nearest_center(data[farthest, ], centers)
        assignments[farthest] <- c
        counts[c] <- 1L
        sums[[c]] <- as.double(data[farthest, ])
        if (origin != c && counts[origin] > 0L) {
          counts[origin] <- counts[origin] - 1L
          sums[[origin]] <- sums[[origin]] - data[farthest, ]
        }
      }
      centers[[c]] <- sums[[c]] / counts[c]
    }
    iter <- iter + 1L
  }
  mean_row <- colMeans(data)
  size <- integer(k)
  withinss <- numeric(k)
  totss <- 0
  for (i in seq_len(n)) {
    size[assignments[i]] <- size[assignments[i]] + 1L
    withinss[assignments[i]] <- withinss[assignments[i]] + sum((data[i, ] - centers[[assignments[i]]])^2)
    totss <- totss + sum((data[i, ] - mean_row)^2)
  }
  totwithinss <- sum(withinss)
  reordered <- reorder_kmeans(assignments, centers, size, withinss)
  list(
    cluster = reordered$cluster,
    centers = reordered$centers,
    totss = totss,
    withinss = reordered$withinss,
    totwithinss = totwithinss,
    betweenss = totss - totwithinss,
    size = reordered$size,
    iter = as.double(iter),
    ifault = 0
  )
}

kmeans_stats <- function(rows, k, nstart = 1L, itermax = 10L, seed = 1L) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  set.seed(as.integer(seed))
  km <- stats::kmeans(m, centers = as.integer(k), nstart = as.integer(nstart), iter.max = as.integer(itermax))
  centers <- lapply(seq_len(nrow(km$centers)), function(i) as.double(km$centers[i, ]))
  list(
    cluster = as.integer(unname(km$cluster)),
    centers = centers,
    totss = as.double(km$totss),
    withinss = as.double(km$withinss),
    totwithinss = as.double(km$tot.withinss),
    betweenss = as.double(km$betweenss),
    size = as.integer(km$size),
    iter = as.double(km$iter),
    ifault = as.integer(km$ifault)
  )
}

orient_cluster <- function(a, b) {
  if (a$min_leaf < b$min_leaf) return(list(a, b))
  if (b$min_leaf < a$min_leaf) return(list(b, a))
  if (a$rid <= b$rid) return(list(a, b))
  list(b, a)
}

merged_centroid <- function(a, b, method) {
  if (method == "median") {
    return((a$centroid + b$centroid) / 2)
  }
  total <- a$size + b$size
  (a$size * a$centroid + b$size * b$centroid) / total
}

updated_distance <- function(method, other, a, b, dik, djk, dij) {
  if (method == "single") return(min(dik, djk))
  if (method == "complete") return(max(dik, djk))
  if (method == "average") return((a$size * dik + b$size * djk) / (a$size + b$size))
  if (method == "mcquitty") return(0.5 * dik + 0.5 * djk)
  if (method %in% c("centroid", "median")) {
    merged <- merged_centroid(a, b, method)
    return(sqrt(sum((other$centroid - merged)^2)))
  }
  merged <- merged_centroid(a, b, "average")
  d <- sqrt(sum((other$centroid - merged)^2))
  if (method == "ward.d2") return(d * d)
  d
}

tie_break_pair <- function(a1, b1, a2, b2) {
  p1 <- orient_cluster(a1, b1)
  p2 <- orient_cluster(a2, b2)
  if (p1[[1]]$min_leaf != p2[[1]]$min_leaf) return(p1[[1]]$min_leaf < p2[[1]]$min_leaf)
  p1[[2]]$min_leaf < p2[[2]]$min_leaf
}

cut_tree_from_result <- function(tree, k = NULL, height_cut = NULL) {
  n <- length(tree$labels)
  parent <- seq_len(n)
  find_root <- function(x) {
    while (parent[x] != x) {
      parent[x] <<- parent[parent[x]]
      x <- parent[x]
    }
    x
  }
  union_nodes <- function(a, b) {
    ra <- find_root(a)
    rb <- find_root(b)
    if (ra == rb) return()
    if (ra < rb) parent[rb] <<- ra else parent[ra] <<- rb
  }
  nodes <- new.env(parent = emptyenv())
  for (i in seq_len(n)) assign(as.character(-i), c(i), envir = nodes)
  merges_to_apply <- if (!is.null(k)) n - as.integer(k) else NULL
  for (step in seq_along(tree$merge)) {
    row <- tree$merge[[step]]
    left <- get(as.character(row[[1]]), envir = nodes)
    right <- get(as.character(row[[2]]), envir = nodes)
    combined <- c(left, right)
    assign(as.character(step), combined, envir = nodes)
    include <- if (!is.null(merges_to_apply)) (step - 1L) < merges_to_apply else tree$height[step] <= height_cut
    if (include && length(combined) > 1L) {
      base <- combined[[1]]
      for (member in combined[-1]) union_nodes(base, member)
    }
  }
  label_map <- list()
  next_label <- 1L
  out <- integer(n)
  for (i in seq_len(n)) {
    root <- find_root(i)
    key <- as.character(root)
    if (is.null(label_map[[key]])) {
      label_map[[key]] <- next_label
      next_label <- next_label + 1L
    }
    out[i] <- label_map[[key]]
  }
  out
}

hclust_stats <- function(rows, method, k = NULL, h = NULL) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  hc <- stats::hclust(stats::dist(m, method = "euclidean"), method = as.character(method))
  labels <- hc$labels
  if (is.null(labels)) labels <- as.character(seq_len(nrow(m)))
  out <- list(
    merge = lapply(seq_len(nrow(hc$merge)), function(i) as.integer(hc$merge[i, ])),
    height = as.double(hc$height),
    order = as.integer(hc$order),
    labels = as.character(labels)
  )
  if (!is.null(k)) out$cut_k <- as.integer(stats::cutree(hc, k = as.integer(k)))
  if (!is.null(h)) out$cut_h <- as.integer(stats::cutree(hc, h = as.double(h)))
  out
}

dbscan_stats <- function(rows, eps, min_pts) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  fit <- dbscan::dbscan(m, eps = as.double(eps), minPts = as.integer(min_pts))
  list(cluster = as.integer(fit$cluster), isseed = as.logical(dbscan::is.corepoint(m, eps = as.double(eps), minPts = as.integer(min_pts))))
}

silhouette_stats <- function(rows, labels) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  sil <- cluster::silhouette(as.integer(unlist(labels)), stats::dist(m, method = "euclidean"))
  raw <- unclass(sil)
  list(
    points = lapply(seq_len(nrow(raw)), function(i) as.double(raw[i, ])),
    avg_width = as.double(summary(sil)$avg.width)
  )
}

knn_neighbors_stats <- function(train_rows, test_rows, k) {
  train <- do.call(rbind, lapply(train_rows, function(r) as.double(unlist(r))))
  test <- do.call(rbind, lapply(test_rows, function(r) as.double(unlist(r))))
  idx_out <- vector("list", nrow(test))
  dist_out <- vector("list", nrow(test))
  for (i in seq_len(nrow(test))) {
    d <- sqrt(rowSums((t(t(train) - test[i, ]))^2))
    ord <- order(d, seq_along(d))
    picked <- ord[seq_len(k)]
    idx_out[[i]] <- as.integer(picked)
    dist_out[[i]] <- as.double(d[picked])
  }
  list(indices = idx_out, distances = dist_out)
}

knn_class_probs <- function(indices, distances, labels, classes, weighting) {
  weights <- rep(0, length(classes))
  use <- seq_along(indices)
  if (any(abs(distances) <= 1e-12)) {
    use <- which(abs(distances) <= 1e-12)
  }
  for (u in use) {
    w <- 1.0
    if (weighting == "distance") {
      if (abs(distances[u]) <= 1e-12) {
        w <- 1.0
      } else {
        w <- 1.0 / distances[u]
      }
    }
    cls <- labels[indices[u]]
    weights[match(cls, classes)] <- weights[match(cls, classes)] + w
  }
  total <- sum(weights)
  if (total > 0) weights <- weights / total
  weights
}

knn_classify_stats <- function(train_rows, test_rows, labels, k, weighting) {
  labels <- as.character(unlist(labels))
  classes <- sort(unique(labels))   # alphabetical, matches insyra's orderedClasses
  nn <- knn_neighbors_stats(train_rows, test_rows, k)
  probs <- vector("list", length(nn$indices))
  preds <- character(length(nn$indices))
  for (i in seq_along(nn$indices)) {
    p <- knn_class_probs(nn$indices[[i]], nn$distances[[i]], labels, classes, weighting)
    probs[[i]] <- as.double(p)
    # Tie-break = alphabetical first, matching R's which.max convention.
    best <- 1
    for (j in 2:length(classes)) {
      if (p[j] > p[best] && abs(p[j] - p[best]) > 1e-12) {
        best <- j
      }
    }
    preds[[i]] <- classes[[best]]
  }
  list(predictions = as.character(preds), classes = as.character(classes), probabilities = probs)
}

knn_regress_stats <- function(train_rows, test_rows, targets, k, weighting) {
  targets <- as.double(unlist(targets))
  nn <- knn_neighbors_stats(train_rows, test_rows, k)
  preds <- numeric(length(nn$indices))
  for (i in seq_along(nn$indices)) {
    use <- seq_along(nn$indices[[i]])
    if (any(abs(nn$distances[[i]]) <= 1e-12)) {
      use <- which(abs(nn$distances[[i]]) <= 1e-12)
    }
    w <- rep(1.0, length(use))
    if (weighting == "distance") {
      for (j in seq_along(use)) {
        d <- nn$distances[[i]][use[j]]
        if (abs(d) > 1e-12) w[j] <- 1.0 / d
      }
    }
    idx <- nn$indices[[i]][use]
    preds[[i]] <- sum(w * targets[idx]) / sum(w)
  }
  list(predictions = as.double(preds))
}

bartlett_sphericity <- function(rows) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  n <- nrow(m)
  p <- ncol(m)
  corr <- cor(m)
  detv <- det(corr)
  chisq <- -((n - 1) - (2 * p + 5) / 6) * log(detv)
  df <- p * (p - 1) / 2
  pval <- 1 - pchisq(chisq, df)
  list(chi_square = chisq, p_value = pval, df = as.double(df))
}

corr_pair_stat_p <- function(x, y, method) {
  n <- length(x)
  if (method == "pearson") {
    r <- cor(x, y, method = "pearson")
    if (abs(r) >= 0.9999) {
      return(list(stat = r, p = 0))
    }
    inf <- correlation_inference(r, n)
    return(list(stat = r, p = inf$p))
  }
  if (method == "spearman") {
    # Match insyra (and R's cor.test): use prho-based exact/AS-89 path when
    # there are no ties; fall back to Fisher r-to-t with ties.
    res <- suppressWarnings(cor.test(x, y, method = "spearman"))
    return(list(stat = unname(res$estimate), p = res$p.value))
  }
  tau <- cor(x, y, method = "kendall")
  if (n <= 7) {
    permute_all <- function(v) {
      if (length(v) == 1) {
        return(list(v))
      }
      res <- list()
      idx <- 1
      for (i in seq_along(v)) {
        sub <- permute_all(v[-i])
        for (p in sub) {
          res[[idx]] <- c(v[i], p)
          idx <- idx + 1
        }
      }
      res
    }
    y_sorted <- sort(as.double(y))
    perms <- permute_all(y_sorted)
    obs <- abs(tau)
    extreme <- 0
    total <- length(perms)
    for (perm in perms) {
      alt_tau <- cor(x, perm, method = "kendall")
      if (abs(alt_tau) >= obs) {
        extreme <- extreme + 1
      }
    }
    p <- extreme / total
  } else {
    z <- 3 * tau * sqrt(n * (n - 1)) / sqrt(2 * (2 * n + 5))
    p <- 2 * (1 - pnorm(abs(z)))
  }
  list(stat = tau, p = p)
}

corr_matrices <- function(rows, method) {
  m <- do.call(rbind, lapply(rows, function(r) as.double(unlist(r))))
  n <- ncol(m)
  corr <- matrix(0, n, n)
  pmat <- matrix(0, n, n)
  for (i in seq_len(n)) {
    for (j in i:n) {
      if (i == j) {
        corr[i, j] <- 1
        pmat[i, j] <- 0
      } else {
        cp <- corr_pair_stat_p(m[, i], m[, j], method)
        corr[i, j] <- cp$stat
        corr[j, i] <- cp$stat
        pmat[i, j] <- cp$p
        pmat[j, i] <- cp$p
      }
    }
  }
  corr_rows <- lapply(seq_len(nrow(corr)), function(i) as.double(corr[i, ]))
  p_rows <- lapply(seq_len(nrow(pmat)), function(i) as.double(pmat[i, ]))
  list(corr = corr_rows, pmat = p_rows)
}

method <- commandArgs(trailingOnly = TRUE)[1]
payload <- fromJSON(commandArgs(trailingOnly = TRUE)[2], simplifyVector = FALSE)

if (method == "single_t") {
  x <- as.double(unlist(payload$x))
  mu <- as.double(payload$mu)
  cl <- as.double(payload$cl)
  n <- length(x)
  mean_x <- mean(x)
  sd_x <- sd(x)
  se <- sd_x / sqrt(n)
  t <- (mean_x - mu) / se
  p <- 2 * (1 - pt(abs(t), n - 1))
  tcrit <- qt(1 - (1 - cl) / 2, n - 1)
  ci <- c(mean_x - tcrit * se, mean_x + tcrit * se)
  out <- list(stat = t, p = p, df = as.double(n - 1), ci = ci, mean = mean_x, effect = (mean_x - mu) / sd_x)
} else if (method == "two_t") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  eq <- isTRUE(payload$equal_var)
  cl <- as.double(payload$cl)
  n1 <- length(x)
  n2 <- length(y)
  m1 <- mean(x)
  m2 <- mean(y)
  v1 <- var(x)
  v2 <- var(y)
  diff <- m1 - m2
  if (eq) {
    pvar <- (((n1 - 1) * v1) + ((n2 - 1) * v2)) / (n1 + n2 - 2)
    se <- sqrt(pvar * (1 / n1 + 1 / n2))
    df <- as.double(n1 + n2 - 2)
    d <- diff / sqrt(pvar)
  } else {
    se <- sqrt(v1 / n1 + v2 / n2)
    df <- (v1 / n1 + v2 / n2)^2 / (((v1 / n1)^2) / (n1 - 1) + ((v2 / n2)^2) / (n2 - 1))
    d <- diff / sqrt((v1 + v2) / 2)
  }
  t <- diff / se
  p <- 2 * (1 - pt(abs(t), df))
  tcrit <- qt(1 - (1 - cl) / 2, df)
  ci <- c(diff - tcrit * se, diff + tcrit * se)
  out <- list(stat = t, p = p, df = df, ci = ci, mean1 = m1, mean2 = m2, effect = d)
} else if (method == "paired_t") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  cl <- as.double(payload$cl)
  dxy <- x - y
  n <- length(dxy)
  md <- mean(dxy)
  sd_d <- sd(dxy)
  se <- sd_d / sqrt(n)
  t <- md / se
  p <- 2 * (1 - pt(abs(t), n - 1))
  tcrit <- qt(1 - (1 - cl) / 2, n - 1)
  ci <- c(md - tcrit * se, md + tcrit * se)
  out <- list(stat = t, p = p, df = as.double(n - 1), ci = ci, mean_diff = md, effect = abs(md) / sd_d)
} else if (method == "single_z") {
  x <- as.double(unlist(payload$x))
  mu <- as.double(payload$mu)
  sigma <- as.double(payload$sigma)
  alt <- as.character(payload$alt)
  cl <- as.double(payload$cl)
  n <- length(x)
  mean_x <- mean(x)
  se <- sigma / sqrt(n)
  z <- (mean_x - mu) / se
  if (alt == "greater") {
    p <- 1 - pnorm(z)
  } else if (alt == "less") {
    p <- pnorm(z)
  } else {
    p <- 2 * (1 - pnorm(abs(z)))
  }
  q <- if (alt == "two-sided") qnorm(1 - (1 - cl) / 2) else qnorm(cl)
  margin <- q * se
  ci <- ci_by_alt(mean_x, margin, alt)
  out <- list(stat = z, p = p, ci = ci, mean = mean_x, effect = abs(mean_x - mu) / sigma)
} else if (method == "two_z") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  s1 <- as.double(payload$sigma1)
  s2 <- as.double(payload$sigma2)
  alt <- as.character(payload$alt)
  cl <- as.double(payload$cl)
  n1 <- length(x)
  n2 <- length(y)
  m1 <- mean(x)
  m2 <- mean(y)
  diff <- m1 - m2
  se <- sqrt((s1^2) / n1 + (s2^2) / n2)
  z <- diff / se
  if (alt == "greater") {
    p <- 1 - pnorm(z)
  } else if (alt == "less") {
    p <- pnorm(z)
  } else {
    p <- 2 * (1 - pnorm(abs(z)))
  }
  q <- if (alt == "two-sided") qnorm(1 - (1 - cl) / 2) else qnorm(cl)
  margin <- q * se
  ci <- ci_by_alt(diff, margin, alt)
  pooled_sigma <- sqrt((s1^2 + s2^2) / 2)   # Cohen's d_av
  out <- list(stat = z, p = p, ci = ci, mean1 = m1, mean2 = m2, effect = abs(diff) / pooled_sigma)
} else if (method == "chi_gof") {
  vals <- trimws(as.character(unlist(payload$values)))
  keys <- sort(unique(vals))
  observed <- as.double(sapply(keys, function(k) sum(vals == k)))
  p_exp <- payload$p
  if (is.null(p_exp) || length(p_exp) == 0) {
    p_exp <- rep(1 / length(observed), length(observed))
  }
  p_exp <- as.double(unlist(p_exp))
  if (isTRUE(payload$rescale)) {
    p_exp <- p_exp / sum(p_exp)
  }
  expected <- sum(observed) * p_exp
  chi <- sum((observed - expected)^2 / expected)
  df <- as.double(length(observed) - 1)
  p <- 1 - pchisq(chi, df)
  out <- list(stat = chi, p = p, df = df, observed = observed, expected = expected, keys = keys)
} else if (method == "chi_ind") {
  rows <- trimws(as.character(unlist(payload$rows)))
  cols <- trimws(as.character(unlist(payload$cols)))
  rkeys <- sort(unique(rows))
  ckeys <- sort(unique(cols))
  obs <- matrix(0, nrow = length(rkeys), ncol = length(ckeys))
  rownames(obs) <- rkeys
  colnames(obs) <- ckeys
  for (i in seq_along(rows)) {
    obs[rows[i], cols[i]] <- obs[rows[i], cols[i]] + 1
  }
  rs <- rowSums(obs)
  cs <- colSums(obs)
  total <- sum(obs)
  exp <- rs %*% t(cs) / total
  chi <- sum((obs - exp)^2 / exp)
  df <- as.double((nrow(obs) - 1) * (ncol(obs) - 1))
  p <- 1 - pchisq(chi, df)
  out <- list(
    stat = chi,
    p = p,
    df = df,
    observed = lapply(seq_len(nrow(obs)), function(i) as.double(obs[i, ])),
    expected = lapply(seq_len(nrow(exp)), function(i) as.double(exp[i, ])),
    row_keys = as.character(rownames(obs)),
    col_keys = as.character(colnames(obs))
  )
} else if (method == "oneway_anova") {
  st <- one_way_stats(payload$groups)
  out <- list(ssb = st$ssb, ssw = st$ssw, dfb = st$dfb, dfw = st$dfw, f = st$f, p = st$p, eta = st$eta, total_ss = st$ssb + st$ssw)
} else if (method == "twoway_anova") {
  out <- two_way_stats(as.integer(payload$a_levels), as.integer(payload$b_levels), payload$cells)
} else if (method == "rm_anova") {
  out <- rm_stats(payload$subjects)
} else if (method == "f_var") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  v1 <- var(x)
  v2 <- var(y)
  f <- if (v1 > v2) v1 / v2 else v2 / v1
  df1 <- as.double(length(x) - 1)
  df2 <- as.double(length(y) - 1)
  cdf <- pf(f, df1, df2)
  p <- 2 * min(cdf, 1 - cdf)
  out <- list(stat = f, p = p, df1 = df1, df2 = df2)
} else if (method == "levene") {
  groups <- lapply(payload$groups, function(g) as.double(unlist(g)))
  dev_groups <- lapply(groups, function(g) abs(g - median(g)))
  st <- one_way_stats(dev_groups)
  out <- list(stat = st$f, p = st$p, df1 = st$dfb, df2 = st$dfw)
} else if (method == "bartlett") {
  groups <- lapply(payload$groups, function(g) as.double(unlist(g)))
  k <- length(groups)
  sum_n1 <- 0
  pooled_log_var <- 0
  weight <- 0
  for (g in groups) {
    n <- length(g)
    v <- var(g)
    if (n < 2 || v <= 0) next
    sum_n1 <- sum_n1 + (n - 1)
    pooled_log_var <- pooled_log_var + (n - 1) * log(v)
    weight <- weight + 1 / (n - 1)
  }
  mean_var <- 0
  for (g in groups) {
    if (length(g) >= 2) {
      mean_var <- mean_var + (length(g) - 1) * var(g)
    }
  }
  mean_var <- mean_var / sum_n1
  T <- (sum_n1 * log(mean_var)) - pooled_log_var
  correction <- 1 + (1 / (3 * (k - 1))) * (weight - 1 / sum_n1)
  chi <- T / correction
  df <- as.double(k - 1)
  p <- 1 - pchisq(chi, df)
  out <- list(stat = chi, p = p, df = df)
} else if (method == "f_reg") {
  ssr <- as.double(payload$ssr)
  sse <- as.double(payload$sse)
  df1 <- as.double(payload$df1)
  df2 <- as.double(payload$df2)
  f <- (ssr / df1) / (sse / df2)
  p <- 1 - pf(f, df1, df2)
  out <- list(stat = f, p = p, df1 = df1, df2 = df2)
} else if (method == "f_nested") {
  rss_r <- as.double(payload$rss_reduced)
  rss_f <- as.double(payload$rss_full)
  df_r <- as.integer(payload$df_reduced)
  df_f <- as.integer(payload$df_full)
  ndf <- as.double(df_r - df_f)
  ddf <- as.double(df_f)
  f <- ((rss_r - rss_f) / ndf) / (rss_f / ddf)
  p <- 1 - pf(f, ndf, ddf)
  out <- list(stat = f, p = p, df1 = ndf, df2 = ddf)
} else if (method == "covariance") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  out <- list(cov = cov(x, y))
} else if (method == "correlation") {
  x <- as.double(unlist(payload$x))
  y <- as.double(unlist(payload$y))
  m <- as.character(payload$corr_method)
  n <- length(x)
  if (m == "pearson") {
    r <- cor(x, y, method = "pearson")
    if (abs(r) >= 0.9999) {
      out <- list(stat = r, p = 0, df = as.double(n - 2))
    } else {
      inf <- correlation_inference(r, n)
      out <- list(stat = r, p = inf$p, df = inf$df, ci = inf$ci)
    }
  } else if (m == "spearman") {
    # Match insyra's prho-based path (R cor.test default).
    res <- suppressWarnings(cor.test(x, y, method = "spearman"))
    rho <- unname(res$estimate)
    p   <- res$p.value
    # CI is Fisher-z (insyra convention; R doesn't return one).
    inf <- correlation_inference(rho, n)
    out <- list(stat = rho, p = p, df = as.double(n - 2), ci = inf$ci)
  } else {
    cp <- corr_pair_stat_p(x, y, "kendall")
    out <- list(stat = cp$stat, p = cp$p)
  }
} else if (method == "bartlett_sphericity") {
  out <- bartlett_sphericity(payload$rows)
} else if (method == "corr_matrix") {
  cm <- corr_matrices(payload$rows, as.character(payload$corr_method))
  out <- list(corr_matrix = cm$corr, p_matrix = cm$pmat)
} else if (method == "corr_analysis") {
  cm <- corr_matrices(payload$rows, as.character(payload$corr_method))
  if (as.character(payload$corr_method) == "pearson") {
    b <- bartlett_sphericity(payload$rows)
    out <- list(corr_matrix = cm$corr, p_matrix = cm$pmat, chi_square = b$chi_square, p_value = b$p_value, df = b$df)
  } else {
    out <- list(corr_matrix = cm$corr, p_matrix = cm$pmat, chi_square = NaN, p_value = NaN, df = 0)
  }
} else if (method == "linear_reg") {
  y <- as.double(unlist(payload$y))
  xs <- lapply(payload$xs, function(v) as.double(unlist(v)))
  X <- cbind(1, do.call(cbind, xs))
  st <- ols_from_matrix(y, X)
  out <- st
  if (length(xs) == 1) {
    out$intercept <- st$coefficients[1]
    out$slope <- st$coefficients[2]
    out$se_intercept <- st$standard_errors[1]
    out$se_slope <- st$standard_errors[2]
    out$t_intercept <- st$t_values[1]
    out$t_slope <- st$t_values[2]
    out$p_intercept <- st$p_values[1]
    out$p_slope <- st$p_values[2]
    out$ci_intercept <- st$confidence_intervals[[1]]
    out$ci_slope <- st$confidence_intervals[[2]]
  }
} else if (method == "poly_reg") {
  y <- as.double(unlist(payload$y))
  x <- as.double(unlist(payload$x))
  degree <- as.integer(payload$degree)
  cols <- lapply(0:degree, function(d) x^d)
  X <- do.call(cbind, cols)
  out <- ols_from_matrix(y, X)
} else if (method == "exp_reg") {
  y <- as.double(unlist(payload$y))
  x <- as.double(unlist(payload$x))
  lny <- log(y)
  X <- cbind(1, x)
  st <- ols_from_matrix(lny, X)
  ln_a <- st$coefficients[1]
  b <- st$coefficients[2]
  a <- exp(ln_a)
  pred <- a * exp(b * x)
  residuals <- y - pred
  sse <- sum(residuals^2)
  sst <- sum((y - mean(y))^2)
  r2 <- 1 - sse / sst
  n <- length(x)
  df <- n - 2
  adj <- 1 - (1 - r2) * ((n - 1) / df)
  mse_log <- sum((lny - (ln_a + b * x))^2) / df
  mean_x <- mean(x)
  sxx <- sum((x - mean_x)^2)
  se_b <- sqrt(mse_log / sxx)
  se_ln_a <- sqrt(mse_log * (1 / n + mean_x^2 / sxx))
  se_a <- a * se_ln_a
  t_a <- a / se_a
  t_b <- b / se_b
  p_a <- 2 * (1 - pt(abs(t_a), df))
  p_b <- 2 * (1 - pt(abs(t_b), df))
  tcrit <- qt(0.975, df)
  out <- list(
    intercept = a,
    slope = b,
    residuals = as.double(residuals),
    r_squared = r2,
    adj_r_squared = adj,
    se_intercept = se_a,
    se_slope = se_b,
    t_intercept = t_a,
    t_slope = t_b,
    p_intercept = p_a,
    p_slope = p_b,
    ci_intercept = c(a - tcrit * se_a, a + tcrit * se_a),
    ci_slope = c(b - tcrit * se_b, b + tcrit * se_b)
  )
} else if (method == "log_reg") {
  y <- as.double(unlist(payload$y))
  x <- as.double(unlist(payload$x))
  lx <- log(x)
  X <- cbind(1, lx)
  st <- ols_from_matrix(y, X)
  a <- st$coefficients[1]
  b <- st$coefficients[2]
  residuals <- as.double(st$residuals)
  sse <- sum(residuals^2)
  n <- length(x)
  df <- n - 2
  mse <- sse / df
  mean_lx <- mean(lx)
  sxx <- sum((lx - mean_lx)^2)
  se_b <- sqrt(mse / sxx)
  se_a <- sqrt(mse * (1 / n + mean_lx^2 / sxx))
  t_a <- a / se_a
  t_b <- b / se_b
  p_a <- 2 * (1 - pt(abs(t_a), df))
  p_b <- 2 * (1 - pt(abs(t_b), df))
  tcrit <- qt(0.975, df)
  out <- list(
    intercept = a,
    slope = b,
    residuals = residuals,
    r_squared = st$r_squared,
    adj_r_squared = st$adj_r_squared,
    se_intercept = se_a,
    se_slope = se_b,
    t_intercept = t_a,
    t_slope = t_b,
    p_intercept = p_a,
    p_slope = p_b,
    ci_intercept = c(a - tcrit * se_a, a + tcrit * se_a),
    ci_slope = c(b - tcrit * se_b, b + tcrit * se_b)
  )
} else if (method == "pca") {
  out <- pca_stats(payload$rows, payload$n_components)
} else if (method == "factor_analysis") {
  out <- factor_analysis_stats(payload$rows, payload$extraction, payload$rotation, payload$scoring, payload$nfactors)
} else if (method == "kmeans") {
  out <- kmeans_stats(payload$rows, payload$k, payload$nstart, payload$itermax, payload$seed)
} else if (method == "hclust") {
  out <- hclust_stats(payload$rows, payload$method, payload$k, payload$h)
} else if (method == "dbscan") {
  out <- dbscan_stats(payload$rows, payload$eps, payload$min_pts)
} else if (method == "silhouette") {
  out <- silhouette_stats(payload$rows, payload$labels)
} else if (method == "knn_classify") {
  out <- knn_classify_stats(payload$train_rows, payload$test_rows, payload$labels, as.integer(payload$k), as.character(payload$weighting))
} else if (method == "knn_regress") {
  out <- knn_regress_stats(payload$train_rows, payload$test_rows, payload$targets, as.integer(payload$k), as.character(payload$weighting))
} else if (method == "knn_neighbors") {
  out <- knn_neighbors_stats(payload$train_rows, payload$test_rows, as.integer(payload$k))
} else if (method == "moment") {
  x <- as.double(unlist(payload$x))
  order <- as.integer(payload$order)
  central <- isTRUE(payload$central)
  if (central) {
    mu <- mean(x)
    out <- list(value = mean((x - mu)^order))
  } else {
    out <- list(value = mean(x^order))
  }
} else if (method == "skewness") {
  x <- as.double(unlist(payload$x))
  mode <- as.character(payload$mode)
  n <- as.double(length(x))
  mu <- mean(x)
  m2 <- mean((x - mu)^2)
  m3 <- mean((x - mu)^3)
  g1 <- if (m2 == 0) 0 else m3 / (m2^(3 / 2))
  if (mode == "g1") {
    out <- list(value = g1)
  } else if (mode == "adjusted") {
    out <- list(value = g1 * sqrt(n * (n - 1)) / (n - 2))
  } else {
    out <- list(value = g1 * (((n - 1) / n)^(3 / 2)))
  }
} else if (method == "kurtosis") {
  x <- as.double(unlist(payload$x))
  mode <- as.character(payload$mode)
  n <- as.double(length(x))
  mu <- mean(x)
  m2 <- mean((x - mu)^2)
  m4 <- mean((x - mu)^4)
  g2 <- m4 / (m2 * m2) - 3
  if (mode == "g2") {
    out <- list(value = g2)
  } else if (mode == "adjusted") {
    out <- list(value = ((g2 * (n + 1) + 6) * (n - 1)) / ((n - 2) * (n - 3)))
  } else {
    out <- list(value = (g2 + 3) * ((1 - 1 / n)^2) - 3)
  }
} else {
  stop(paste("unsupported method:", method))
}

cat(toJSON(out, auto_unbox = TRUE, digits = 16))
