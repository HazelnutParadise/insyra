# psych::fac (FULL) — 包含 fit.residuals / FAgr.minres / FAout / FA.OLS 等內部函數
# dumped at 2025-10-04 13:16:30.648784
fac <- 
function (r, nfactors = 1, n.obs = NA, rotate = "oblimin", scores = "tenBerge", 
    residuals = FALSE, SMC = TRUE, covar = FALSE, missing = FALSE, 
    impute = "median", min.err = 0.001, max.iter = 50, symmetric = TRUE, 
    warnings = TRUE, fm = "minres", alpha = 0.1, oblique.scores = FALSE, 
    np.obs = NULL, use = "pairwise", cor = "cor", correct = 0.5, 
    weight = NULL, n.rotations = 1, hyper = 0.15, smooth = TRUE, 
    ...) 
{
    cl <- match.call()
    control <- NULL
    "fit.residuals" <- function(Psi, S, nf, S.inv = NULL, fm) {
        diag(S) <- 1 - Psi
        if (!is.null(S.inv)) 
            sd.inv <- diag(1/diag(S.inv))
        eigens <- eigen(S)
        eigens$values[eigens$values < .Machine$double.eps] <- 100 * 
            .Machine$double.eps
        if (nf > 1) {
            loadings <- eigens$vectors[, 1:nf] %*% diag(sqrt(eigens$values[1:nf]))
        }
        else {
            loadings <- eigens$vectors[, 1] * sqrt(eigens$values[1])
        }
        model <- loadings %*% t(loadings)
        switch(fm, wls = {
            residual <- sd.inv %*% (S - model)^2 %*% sd.inv
        }, gls = {
            residual <- (S.inv %*% (S - model))^2
        }, uls = {
            residual <- (S - model)^2
        }, ols = {
            residual <- (S - model)
            residual <- residual[lower.tri(residual)]
            residual <- residual^2
        }, minres = {
            residual <- (S - model)
            residual <- residual[lower.tri(residual)]
            residual <- residual^2
        }, old.min = {
            residual <- (S - model)
            residual <- residual[lower.tri(residual)]
            residual <- residual^2
        }, minchi = {
            residual <- (S - model)^2
            residual <- residual * np.obs
            diag(residual) <- 0
        })
        error <- sum(residual)
    }
    "fit" <- function(S, nf, fm, covar) {
        if (is.logical(SMC)) {
            S.smc <- smc(S, covar)
        }
        else {
            S.smc <- SMC
        }
        upper <- max(S.smc, 1)
        if ((fm == "wls") | (fm == "gls")) {
            S.inv <- solve(S)
        }
        else {
            S.inv <- NULL
        }
        if (!covar && (sum(S.smc) == nf) && (nf > 1)) {
            start <- rep(0.5, nf)
        }
        else {
            start <- diag(S) - S.smc
        }
        if (fm == "ml" || fm == "mle") {
            res <- optim(start, FAfn, FAgr, method = "L-BFGS-B", 
                lower = 0.005, upper = upper, control = c(list(fnscale = 1, 
                  parscale = rep(0.01, length(start))), control), 
                nf = nf, S = S)
        }
        else {
            if (fm == "ols") {
                if (is.logical(SMC)) {
                  start <- diag(S) - smc(S, covar)
                }
                else {
                  start <- SMC
                }
                res <- optim(start, FA.OLS, method = "L-BFGS-B", 
                  lower = 0.005, upper = upper, control = c(list(fnscale = 1, 
                    parscale = rep(0.01, length(start)))), nf = nf, 
                  S = S)
            }
            else {
                if ((fm == "minres") | (fm == "uls")) {
                  start <- diag(S) - smc(S, covar)
                  res <- optim(start, fit.residuals, gr = FAgr.minres, 
                    method = "L-BFGS-B", lower = 0.005, upper = upper, 
                    control = c(list(fnscale = 1, parscale = rep(0.01, 
                      length(start)))), nf = nf, S = S, fm = fm)
                }
                else {
                  start <- smc(S, covar)
                  res <- optim(start, fit.residuals, gr = FAgr.minres2, 
                    method = "L-BFGS-B", lower = 0.005, upper = upper, 
                    control = c(list(fnscale = 1, parscale = rep(0.01, 
                      length(start)))), nf = nf, S = S, S.inv = S.inv, 
                    fm = fm)
                }
            }
        }
        if ((fm == "wls") | (fm == "gls") | (fm == "ols") | (fm == 
            "uls") | (fm == "minres") | (fm == "old.min")) {
            Lambda <- FAout.wls(res$par, S, nf)
        }
        else {
            Lambda <- FAout(res$par, S, nf)
        }
        result <- list(loadings = Lambda, res = res, S = S)
    }
    FAfn <- function(Psi, S, nf) {
        sc <- diag(1/sqrt(Psi))
        Sstar <- sc %*% S %*% sc
        E <- eigen(Sstar, symmetric = TRUE, only.values = TRUE)
        e <- E$values[-(1:nf)]
        e <- sum(log(e) - e) - nf + nrow(S)
        -e
    }
    FAgr <- function(Psi, S, nf) {
        sc <- diag(1/sqrt(Psi))
        Sstar <- sc %*% S %*% sc
        E <- eigen(Sstar, symmetric = TRUE)
        L <- E$vectors[, 1:nf, drop = FALSE]
        load <- L %*% diag(sqrt(pmax(E$values[1:nf] - 1, 0)), 
            nf)
        load <- diag(sqrt(Psi)) %*% load
        g <- load %*% t(load) + diag(Psi) - S
        diag(g)/Psi^2
    }
    FAgr.minres2 <- function(Psi, S, nf, S.inv, fm) {
        sc <- diag(1/sqrt(Psi))
        Sstar <- sc %*% S %*% sc
        E <- eigen(Sstar, symmetric = TRUE)
        L <- E$vectors[, 1:nf, drop = FALSE]
        load <- L %*% diag(sqrt(pmax(E$values[1:nf] - 1, 0)), 
            nf)
        load <- diag(sqrt(Psi)) %*% load
        g <- load %*% t(load) + diag(Psi) - S
        if (fm == "minchi") {
            g <- g * np.obs
        }
        diag(g)/Psi^2
    }
    FAgr.minres <- function(Psi, S, nf, fm) {
        Sstar <- S - diag(Psi)
        E <- eigen(Sstar, symmetric = TRUE)
        L <- E$vectors[, 1:nf, drop = FALSE]
        load <- L %*% diag(sqrt(pmax(E$values[1:nf], 0)), nf)
        g <- load %*% t(load) + diag(Psi) - S
        diag(g)
    }
    FAout <- function(Psi, S, q) {
        sc <- diag(1/sqrt(Psi))
        Sstar <- sc %*% S %*% sc
        E <- eigen(Sstar, symmetric = TRUE)
        L <- E$vectors[, 1L:q, drop = FALSE]
        load <- L %*% diag(sqrt(pmax(E$values[1L:q] - 1, 0)), 
            q)
        diag(sqrt(Psi)) %*% load
    }
    FAout.wls <- function(Psi, S, q) {
        diag(S) <- diag(S) - Psi
        E <- eigen(S, symmetric = TRUE)
        L <- E$vectors[, 1L:q, drop = FALSE] %*% diag(sqrt(pmax(E$values[1L:q, 
            drop = FALSE], 0)), q)
        return(L)
    }
    "MRFA" <- function(S, nf) {
        com.glb <- glb.algebraic(S)
        L <- FAout.wls(1 - com.glb$solution, S, nf)
        h2 <- com.glb$solution
        result <- list(loadings = L, communality = h2)
    }
    FA.OLS <- function(Psi, S, nf) {
        E <- eigen(S - diag(Psi), symmetric = T)
        U <- E$vectors[, 1:nf, drop = FALSE]
        D <- E$values[1:nf, drop = FALSE]
        D[D < 0] <- 0
        if (length(D) < 2) {
            L <- U * sqrt(D)
        }
        else {
            L <- U %*% diag(sqrt(D))
        }
        model <- L %*% t(L)
        diag(model) <- diag(S)
        return(sum((S - model)^2)/2)
    }
    FAgr.OLS <- function(Psi, S, nf) {
        E <- eigen(S - diag(Psi), symmetric = TRUE)
        U <- E$vectors[, 1:nf, drop = FALSE]
        D <- E$values[1:nf]
        D[D < 0] <- 0
        L <- U %*% diag(sqrt(D))
        model <- L %*% t(L)
        g <- diag(Psi) - diag(S - model)
        diag(g)/Psi^2
    }
    if (fm == "mle" || fm == "MLE" || fm == "ML") 
        fm <- "ml"
    if (!any(fm %in% (c("pa", "alpha", "minrank", "wls", "gls", 
        "minres", "minchi", "uls", "ml", "mle", "ols", "old.min")))) {
        message("factor method not specified correctly, minimum residual (unweighted least squares  used")
        fm <- "minres"
    }
    x.matrix <- r
    n <- dim(r)[2]
    if (!isCorrelation(r) & !isCovariance(r)) {
        matrix.input <- FALSE
        n.obs <- dim(r)[1]
        if (missing) {
            x.matrix <- as.matrix(x.matrix)
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
        np.obs <- pairwiseCount(r)
        if (covar) {
            cor <- "cov"
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
        matrix.input <- TRUE
        if (fm == "minchi") {
            if (is.null(np.obs)) {
                fm <- "minres"
                message("factor method minchi does not make sense unless we know the sample size, minres used instead")
            }
        }
        if (is.na(n.obs) && !is.null(np.obs)) 
            n.obs <- max(as.vector(np.obs))
        if (!is.matrix(r)) {
            r <- as.matrix(r)
        }
        if (!covar) {
            r <- cov2cor(r)
        }
    }
    if (!residuals) {
        result <- list(values = c(rep(0, n)), rotation = rotate, 
            n.obs = n.obs, np.obs = np.obs, communality = c(rep(0, 
                n)), loadings = matrix(rep(0, n * n), ncol = n), 
            fit = 0)
    }
    else {
        result <- list(values = c(rep(0, n)), rotation = rotate, 
            n.obs = n.obs, np.obs = np.obs, communality = c(rep(0, 
                n)), loadings = matrix(rep(0, n * n), ncol = n), 
            residual = matrix(rep(0, n * n), ncol = n), fit = 0, 
            r = r)
    }
    if (is.null(SMC)) 
        SMC = TRUE
    r.mat <- r
    Phi <- NULL
    colnames(r.mat) <- rownames(r.mat) <- colnames(r)
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
    if (is.logical(SMC)) {
        if (SMC) {
            if (nfactors <= n) {
                diag(r.mat) <- smc(r, covar = covar)
            }
            else {
                if (warnings) {
                  message("In fa, too many factors requested for this number of variables to use SMC for communality estimates, 1s are used instead")
                }
            }
        }
        else {
            diag(r.mat) <- 1
        }
    }
    else {
        diag(r.mat) <- SMC
    }
    orig <- diag(r)
    comm <- sum(diag(r.mat))
    err <- comm
    i <- 1
    comm.list <- list()
    if (fm == "alpha") {
        i <- 1
        e.values <- eigen(r, symmetric = symmetric)$values
        H2 <- diag(r.mat)
        while (err > min.err) {
            r.mat <- cov2cor(r.mat)
            eigens <- eigen(r.mat, symmetric = symmetric)
            loadings <- eigens$vectors[, 1:nfactors, drop = FALSE] %*% 
                diag(sqrt(eigens$values[1:nfactors, drop = FALSE]))
            model <- loadings %*% t(loadings)
            newH2 <- H2 * diag(model)
            err <- sum(abs(H2 - newH2))
            r.mat <- r
            diag(r.mat) <- newH2
            H2 <- newH2
            i <- i + 1
            if (i > max.iter) {
                if (warnings) {
                  message("maximum iteration exceeded")
                }
                err <- 0
            }
        }
        loadings <- sqrt(H2) * loadings
        eigens <- sqrt(H2) * eigens$values
        comm1 <- sum(H2)
    }
    if (fm == "pa") {
        e.values <- eigen(r, symmetric = symmetric)$values
        while (err > min.err) {
            eigens <- eigen(r.mat, symmetric = symmetric)
            if (nfactors > 1) {
                loadings <- eigens$vectors[, 1:nfactors] %*% 
                  diag(sqrt(eigens$values[1:nfactors]))
            }
            else {
                loadings <- eigens$vectors[, 1] * sqrt(eigens$values[1])
            }
            model <- loadings %*% t(loadings)
            new <- diag(model)
            comm1 <- sum(new)
            diag(r.mat) <- new
            err <- abs(comm - comm1)
            if (is.na(err)) {
                warning("imaginary eigen value condition encountered in fa\n Try again with SMC=FALSE \n exiting fa")
                break
            }
            comm <- comm1
            comm.list[[i]] <- comm1
            i <- i + 1
            if (i > max.iter) {
                if (warnings) {
                  message("maximum iteration exceeded")
                }
                err <- 0
            }
        }
        eigens <- eigens$values
    }
    if (fm == "minrank") {
        mrfa <- MRFA(r, nfactors)
        loadings <- mrfa$loadings
        model <- loadings %*% t(loadings)
        e.values <- eigen(r)$values
        S <- r
        diag(S) <- diag(model)
        eigens <- eigen(S)$values
    }
    if ((fm == "wls") | (fm == "minres") | (fm == "minchi") | 
        (fm == "gls") | (fm == "uls") | (fm == "ml") | (fm == 
        "mle") | (fm == "ols") | (fm == "old.min")) {
        uls <- fit(r, nfactors, fm, covar = covar)
        e.values <- eigen(r)$values
        result.res <- uls$res
        loadings <- uls$loadings
        model <- loadings %*% t(loadings)
        S <- r
        diag(S) <- diag(model)
        eigens <- eigen(S)$values
    }
    if (!is.double(loadings)) {
        warning("the matrix has produced imaginary results -- proceed with caution")
        loadings <- matrix(as.double(loadings), ncol = nfactors)
    }
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
        colnames(loadings) <- "MR1"
    }
    switch(fm, alpha = {
        colnames(loadings) <- paste("alpha", 1:nfactors, sep = "")
    }, wls = {
        colnames(loadings) <- paste("WLS", 1:nfactors, sep = "")
    }, pa = {
        colnames(loadings) <- paste("PA", 1:nfactors, sep = "")
    }, gls = {
        colnames(loadings) <- paste("GLS", 1:nfactors, sep = "")
    }, ml = {
        colnames(loadings) <- paste("ML", 1:nfactors, sep = "")
    }, minres = {
        colnames(loadings) <- paste("MR", 1:nfactors, sep = "")
    }, minrank = {
        colnames(loadings) <- paste("MRFA", 1:nfactors, sep = "")
    }, minchi = {
        colnames(loadings) <- paste("MC", 1:nfactors, sep = "")
    })
    rownames(loadings) <- rownames(r)
    loadings[loadings == 0] <- 10^-15
    model <- loadings %*% t(loadings)
    f.loadings <- loadings
    rot.mat <- NULL
    rotated <- NULL
    if (rotate != "none") {
        if (nfactors > 1) {
            if (n.rotations > 1) {
                rotated <- faRotations(loadings, r = r, n.rotations = n.rotations, 
                  rotate = rotate, hyper = hyper)
                loadings = rotated$loadings
                Phi <- rotated$Phi
                rot.mat = rotated$rot.mat
            }
            else {
                rotated <- NULL
                if (rotate == "varimax" | rotate == "Varimax" | 
                  rotate == "quartimax" | rotate == "bentlerT" | 
                  rotate == "geominT" | rotate == "targetT" | 
                  rotate == "bifactor" | rotate == "TargetT" | 
                  rotate == "equamax" | rotate == "varimin" | 
                  rotate == "specialT" | rotate == "Promax" | 
                  rotate == "promax" | rotate == "cluster" | 
                  rotate == "biquartimin" | rotate == "TargetQ" | 
                  rotate == "specialQ") {
                  Phi <- NULL
                  switch(rotate, varimax = {
                    rotated <- stats::varimax(loadings)
                    loadings <- rotated$loadings
                    rot.mat <- rotated$rotmat
                  }, Varimax = {
                    if (!requireNamespace("GPArotation")) {
                      stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                    }
                    rotated <- GPArotation::Varimax(loadings, 
                      ...)
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
                    rotated <- GPArotation::geominT(loadings, 
                      ...)
                    loadings <- rotated$loadings
                    rot.mat <- t(solve(rotated$Th))
                  }, targetT = {
                    if (!requireNamespace("GPArotation")) {
                      stop("I am sorry, to do this rotation requires the GPArotation package to be installed")
                    }
                    rotated <- GPArotation::targetT(loadings, 
                      Tmat = diag(ncol(loadings)), ...)
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
                    pro <- kaiser(loadings, rotate = "Promax", 
                      ...)
                    loadings <- pro$loadings
                    rot.mat <- pro$rotmat
                    Phi <- pro$Phi
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
                  }, TargetQ = {
                    ob <- TargetQ(loadings, ...)
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
                  }
                }
            }
        }
    }
    else {
        rotated <- NULL
    }
    signed <- sign(colSums(loadings))
    signed[signed == 0] <- 1
    loadings <- loadings %*% diag(signed)
    if (!is.null(Phi)) {
        Phi <- diag(signed) %*% Phi %*% diag(signed)
    }
    switch(fm, alpha = {
        colnames(loadings) <- paste("alpha", 1:nfactors, sep = "")
    }, wls = {
        colnames(loadings) <- paste("WLS", 1:nfactors, sep = "")
    }, pa = {
        colnames(loadings) <- paste("PA", 1:nfactors, sep = "")
    }, gls = {
        colnames(loadings) <- paste("GLS", 1:nfactors, sep = "")
    }, ml = {
        colnames(loadings) <- paste("ML", 1:nfactors, sep = "")
    }, minres = {
        colnames(loadings) <- paste("MR", 1:nfactors, sep = "")
    }, minrank = {
        colnames(loadings) <- paste("MRFA", 1:nfactors, sep = "")
    }, uls = {
        colnames(loadings) <- paste("ULS", 1:nfactors, sep = "")
    }, old.min = {
        colnames(loadings) <- paste0("oldmin", 1:nfactors)
    }, minchi = {
        colnames(loadings) <- paste("MC", 1:nfactors, sep = "")
    })
    if (nfactors > 1) {
        if (is.null(Phi)) {
            ev.rotated <- diag(t(loadings) %*% loadings)
        }
        else {
            ev.rotated <- diag(Phi %*% t(loadings) %*% loadings)
        }
        ev.order <- order(ev.rotated, decreasing = TRUE)
        loadings <- loadings[, ev.order]
    }
    rownames(loadings) <- colnames(r)
    if (!is.null(Phi)) {
        Phi <- Phi[ev.order, ev.order]
    }
    class(loadings) <- "loadings"
    if (nfactors < 1) 
        nfactors <- n
    result <- factor.stats(r, loadings, Phi, n.obs = n.obs, np.obs = np.obs, 
        alpha = alpha, smooth = smooth)
    result$rotation <- rotate
    if (nfactors != 1) {
        result$hyperplane <- colSums(abs(loadings) < hyper)
    }
    else {
        result$hyperplane <- sum(abs(loadings) < hyper)
    }
    result$communality <- diag(model)
    if (max(result$communality > 1) && !covar) 
        warning("An ultra-Heywood case was detected.  Examine the results carefully")
    if (fm == "minrank") {
        result$communalities <- mrfa$communality
    }
    else {
        if (fm == "pa" | fm == "alpha") {
            result$communalities <- comm1
        }
        else {
            result$communalities <- 1 - result.res$par
        }
    }
    result$uniquenesses <- diag(r - model)
    result$values <- eigens
    result$e.values <- e.values
    result$loadings <- loadings
    result$model <- model
    result$fm <- fm
    result$rot.mat <- rot.mat
    if (!is.null(Phi)) {
        colnames(Phi) <- rownames(Phi) <- colnames(loadings)
        result$Phi <- Phi
        Structure <- loadings %*% Phi
    }
    else {
        Structure <- loadings
    }
    class(Structure) <- "loadings"
    result$Structure <- Structure
    if (fm == "pa") 
        result$communality.iterations <- unlist(comm.list)
    result$method = scores
    if (oblique.scores) {
        result$scores <- factor.scores(x.matrix, f = loadings, 
            Phi = NULL, method = scores, missing = missing, impute = impute)
    }
    else {
        result$scores <- factor.scores(x.matrix, f = loadings, 
            Phi = Phi, method = scores, missing = missing, impute = impute)
    }
    if (is.null(result$scores$R2)) 
        result$scores$R2 <- NA
    result$R2.scores <- result$scores$R2
    result$weights <- result$scores$weights
    result$scores <- result$scores$scores
    if (!is.null(result$scores)) 
        colnames(result$scores) <- colnames(loadings)
    result$factors <- nfactors
    result$r <- r
    result$np.obs <- np.obs
    result$fn <- "fa"
    result$fm <- fm
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
    vtotal <- sum(result$communality + result$uniquenesses)
    names(vx) <- colnames(loadings)
    varex <- rbind(`SS loadings` = vx)
    varex <- rbind(varex, `Proportion Var` = vx/vtotal)
    ECV <- varex[2]
    if (nfactors > 1) {
        varex <- rbind(varex, `Cumulative Var` = cumsum(vx/vtotal))
        varex <- rbind(varex, `Proportion Explained` = vx/sum(vx))
        varex <- rbind(varex, `Cumulative Proportion` = cumsum(vx/sum(vx)))
        ECV <- varex[5, ]
    }
    result$rotated <- rotated$rotation.stats
    result$Vaccounted <- varex
    result$ECV <- ECV
    result$Call <- cl
    class(result) <- c("psych", "fa")
    return(result)
}

