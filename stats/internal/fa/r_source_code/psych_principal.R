function (r, nfactors = 1, residuals = FALSE, rotate = "varimax", 
    n.obs = NA, covar = FALSE, scores = TRUE, missing = FALSE, 
    impute = "median", oblique.scores = TRUE, method = "regression", 
    use = "pairwise", cor = "cor", correct = 0.5, weight = NULL, 
    ...) 
{
    cl <- match.call()
    n <- dim(r)[2]
    if (!isCorrelation(r) && (!isCovariance(r))) {
        raw <- TRUE
        n.obs <- dim(r)[1]
        if (scores) {
            x.matrix <- as.matrix(r)
            if (missing) {
                miss <- which(is.na(x.matrix), arr.ind = TRUE)
                if (impute == "mean") {
                  item.means <- colMeans(x.matrix, na.rm = TRUE)
                  x.matrix[miss] <- item.means[miss[, 2]]
                }
                else {
                  item.med <- apply(x.matrix, 2, median, na.rm = TRUE)
                  x.matrix[miss] <- item.med[miss[, 2]]
                }
            }
        }
        switch(cor, cor = {
            if (!is.null(weight)) {
                r <- cor.wt(r, w = weight)$r
            } else {
                r <- cor(r, use = use)
            }
        }, cov = {
            r <- cov(r, use = use)
            covar <- TRUE
        }, wtd = {
            r <- cor.wt(r, w = weight)$r
        }, spearman = {
            r <- cor(r, use = use, method = "spearman")
        }, kendall = {
            r <- cor(r, use = use, method = "kendall")
        }, tet = {
            r <- tetrachoric(r, correct = correct, weight = weight)$rho
        }, poly = {
            r <- polychoric(r, correct = correct, weight = weight)$rho
        }, tetrachoric = {
            r <- tetrachoric(r, correct = correct, weight = weight)$rho
        }, polychoric = {
            r <- polychoric(r, correct = correct, weight = weight)$rho
        }, mixed = {
            r <- mixedCor(r, use = use, correct = correct)$rho
        }, Yuleb = {
            r <- YuleCor(r, , bonett = TRUE)$rho
        }, YuleQ = {
            r <- YuleCor(r, 1)$rho
        }, YuleY = {
            r <- YuleCor(r, 0.5)$rho
        })
    }
    else {
        raw <- FALSE
        if (!is.matrix(r)) {
            r <- as.matrix(r)
        }
        sds <- sqrt(diag(r))
        if (!covar) 
            r <- r/(sds %o% sds)
    }
    if (!residuals) {
        result <- list(values = c(rep(0, n)), rotation = rotate, 
            n.obs = n.obs, communality = c(rep(0, n)), loadings = matrix(rep(0, 
                n * n), ncol = n), fit = 0, fit.off = 0)
    }
    else {
        result <- list(values = c(rep(0, n)), rotation = rotate, 
            n.obs = n.obs, communality = c(rep(0, n)), loadings = matrix(rep(0, 
                n * n), ncol = n), residual = matrix(rep(0, n * 
                n), ncol = n), fit = 0, fit.off = 0)
    }
    if (any(is.na(r))) {
        bad <- TRUE
        tempr <- r
        wcl <- NULL
        while (bad) {
            wc <- table(which(is.na(tempr), arr.ind = TRUE))
            wcl <- c(wcl, as.numeric(names(which(wc == max(wc)))))
            tempr <- r[-wcl, -wcl]
            if (any(is.na(tempr))) {
                bad <- TRUE
            }
            else {
                bad <- FALSE
            }
        }
        cat("\nLikely variables with missing values are ", colnames(r)[wcl], 
            " \n")
        stop("I am sorry: missing values (NAs) in the correlation matrix do not allow me to continue.\nPlease drop those variables and try again.")
    }
    eigens <- eigen(r)
    result$values <- eigens$values
    eigens$values[eigens$values < .Machine$double.eps] <- .Machine$double.eps
    loadings <- eigens$vectors %*% sqrt(diag(eigens$values, nrow = length(eigens$values)))
    if (nfactors > 0) {
        loadings <- loadings[, 1:nfactors]
    }
    else {
        nfactors <- n
    }
    if (nfactors > 1) {
        communalities <- rowSums(loadings^2)
    }
    else {
        communalities <- loadings^2
    }
    uniquenesses <- diag(r) - communalities
    names(communalities) <- colnames(r)
    if (nfactors > 1) {
        sign.tot <- vector(mode = "numeric", length = nfactors)
        sign.tot <- sign(colSums(loadings))
        sign.tot[sign.tot == 0] <- 1
        loadings <- loadings %*% diag(sign.tot)
    }
    else {
        if (sum(loadings) < 0) {
            loadings <- -as.matrix(loadings)
        }
        else {
            loadings <- as.matrix(loadings)
        }
        colnames(loadings) <- "PC1"
    }
    colnames(loadings) <- paste("PC", 1:nfactors, sep = "")
    rownames(loadings) <- rownames(r)
    Phi <- NULL
    rot.mat <- NULL
    if (rotate != "none") {
        if (nfactors > 1) {
            if (rotate == "varimax" | rotate == "Varimax" | rotate == 
                "quartimax" | rotate == "bentlerT" | rotate == 
                "geominT" | rotate == "targetT" | rotate == "bifactor" | 
                rotate == "TargetT" | rotate == "equamax" | rotate == 
                "varimin" | rotate == "specialT" | rotate == 
                "Promax" | rotate == "promax" | rotate == "cluster" | 
                rotate == "biquartimin" | rotate == "specialQ") {
                Phi <- NULL
                colnames(loadings) <- paste("RC", 1:nfactors, 
                  sep = "")
                switch(rotate, varimax = {
                  rotated <- stats::varimax(loadings, ...)
                  loadings <- rotated$loadings
                  rot.mat <- rotated$rotmat
                }, Varimax = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rotated <- GPArotation::Varimax(loadings, ...)
                  loadings <- rotated$loadings
                  rot.mat <- t(solve(rotated$Th))
                }, quartimax = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rotated <- GPArotation::quartimax(loadings, 
                    ...)
                  loadings <- rotated$loadings
                  rot.mat <- t(solve(rotated$Th))
                }, bentlerT = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rotated <- GPArotation::bentlerT(loadings, 
                    ...)
                  loadings <- rotated$loadings
                  rot.mat <- t(solve(rotated$Th))
                }, geominT = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rotated <- GPArotation::geominT(loadings, ...)
                  loadings <- rotated$loadings
                  rot.mat <- t(solve(rotated$Th))
                }, targetT = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rotated <- GPArotation::targetT(loadings, Tmat = diag(ncol(loadings)), 
                    ...)
                  loadings <- rotated$loadings
                  rot.mat <- t(solve(rotated$Th))
                }, bifactor = {
                  rot <- bifactor(loadings, ...)
                  loadings <- rot$loadings
                  rot.mat <- t(solve(rot$Th))
                }, TargetT = {
                  if (!requireNamespace("GPArotation")) {
                    stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                  }
                  rot <- GPArotation::targetT(loadings, Tmat = diag(ncol(loadings)), 
                    ...)
                  loadings <- rot$loadings
                  rot.mat <- t(solve(rot$Th))
                }, equamax = {
                  rot <- equamax(loadings, ...)
                  loadings <- rot$loadings
                  rot.mat <- t(solve(rot$Th))
                }, varimin = {
                  rot <- varimin(loadings, ...)
                  loadings <- rot$loadings
                  rot.mat <- t(solve(rot$Th))
                }, specialT = {
                  rot <- specialT(loadings, ...)
                  loadings <- rot$loadings
                  rot.mat <- t(solve(rot$Th))
                }, Promax = {
                  pro <- Promax(loadings, ...)
                  loadings <- pro$loadings
                  Phi <- pro$Phi
                  rot.mat <- pro$rotmat
                }, promax = {
                  pro <- stats::promax(loadings, ...)
                  loadings <- pro$loadings
                  rot.mat <- pro$rotmat
                  ui <- solve(rot.mat)
                  Phi <- cov2cor(ui %*% t(ui))
                }, cluster = {
                  loadings <- varimax(loadings, ...)$loadings
                  pro <- target.rot(loadings)
                  loadings <- pro$loadings
                  Phi <- pro$Phi
                  rot.mat <- pro$rotmat
                }, biquartimin = {
                  ob <- biquartimin(loadings, ...)
                  loadings <- ob$loadings
                  Phi <- ob$Phi
                  rot.mat <- t(solve(ob$Th))
                }, specialQ = {
                  ob <- specialQ(loadings, ...)
                  loadings <- ob$loadings
                  Phi <- ob$Phi
                  rot.mat <- t(solve(pro$Th))
                })
            }
            else {
                colnames(loadings) <- paste("TC", 1:nfactors, 
                  sep = "")
                if (rotate == "oblimin" | rotate == "quartimin" | 
                  rotate == "simplimax" | rotate == "geominQ" | 
                  rotate == "bentlerQ" | rotate == "targetQ") {
                  if (!requireNamespace("GPArotation")) {
                    warning("I am sorry, to do these rotations requires the GPArotation package to be installed")
                    Phi <- NULL
                  }
                  else {
                    ob <- try(do.call(getFromNamespace(rotate, 
                      "GPArotation"), list(loadings, ...)))
                    if (inherits(ob, as.character("try-error"))) {
                      warning("The requested transformaton failed, Promax was used instead as an oblique transformation")
                      ob <- Promax(loadings)
                    }
                    loadings <- ob$loadings
                    Phi <- ob$Phi
                    rot.mat <- t(solve(ob$Th))
                  }
                }
                else {
                  message("Specified rotation not found, rotate='none' used")
                  colnames(loadings) <- paste("PC", 1:nfactors, 
                    sep = "")
                }
            }
        }
    }
    if (nfactors > 1) {
        ev.rotated <- diag(t(loadings) %*% loadings)
        ev.order <- order(ev.rotated, decreasing = TRUE)
        loadings <- loadings[, ev.order]
    }
    if (!is.null(Phi)) {
        Phi <- Phi[ev.order, ev.order]
    }
    signed <- sign(colSums(loadings))
    c.names <- colnames(loadings)
    signed[signed == 0] <- 1
    loadings <- loadings %*% diag(signed)
    colnames(loadings) <- c.names
    if (!is.null(Phi)) {
        Phi <- diag(signed) %*% Phi %*% diag(signed)
        colnames(Phi) <- rownames(Phi) <- c.names
    }
    class(loadings) <- "loadings"
    if (is.null(Phi)) {
        if (nfactors > 1) {
            vx <- colSums(loadings^2)
        }
        else {
            vx <- sum(loadings^2)
        }
    }
    else {
        vx <- diag(Phi %*% t(loadings) %*% loadings)
    }
    vtotal <- sum(communalities + uniquenesses)
    names(vx) <- colnames(loadings)
    varex <- rbind(`SS loadings` = vx)
    varex <- rbind(varex, `Proportion Var` = vx/vtotal)
    if (nfactors > 1) {
        varex <- rbind(varex, `Cumulative Var` = cumsum(vx/vtotal))
        varex <- rbind(varex, `Proportion Explained` = vx/sum(vx))
        varex <- rbind(varex, `Cumulative Proportion` = cumsum(vx/sum(vx)))
    }
    result$n.obs <- n.obs
    stats <- factor.stats(r, loadings, Phi, n.obs, fm = "pc")
    class(result) <- c("psych", "principal")
    result$fn <- "principal"
    result$loadings <- loadings
    result$Phi <- Phi
    result$Call <- cl
    result$communality <- communalities
    result$uniquenesses <- uniquenesses
    result$complexity <- stats$complexity
    result$valid <- stats$valid
    result$chi <- stats$chi
    result$EPVAL <- stats$EPVAL
    result$R2 <- stats$R2
    result$objective <- stats$objective
    result$residual <- stats$residual
    result$rms <- stats$rms
    result$fit <- stats$fit
    result$fit.off <- stats$fit.off
    result$factors <- stats$factors
    result$dof <- stats$dof
    result$null.dof <- stats$null.dof
    result$null.model <- stats$null.model
    result$criteria <- stats$criteria
    result$STATISTIC <- stats$STATISTIC
    result$PVAL <- stats$PVAL
    result$weights <- stats$weights
    result$r.scores <- stats$r.scores
    result$rot.mat <- rot.mat
    result$Vaccounted <- varex
    if (!is.null(Phi) && oblique.scores) {
        result$Structure <- loadings %*% Phi
    }
    else {
        result$Structure <- loadings
    }
    if (scores && raw) {
        result$weights <- try(solve(r, result$Structure), silent = TRUE)
        if (inherits(result$weights, "try-error")) {
            warning("The matrix is not positive semi-definite, scores found from Structure loadings")
            result$weights <- result$Structure
        }
        result$scores <- scale(x.matrix, scale = !covar) %*% 
            result$weights
    }
    return(result)
}
<bytecode: 0x0000022a139bfa58>
<environment: namespace:psych>
