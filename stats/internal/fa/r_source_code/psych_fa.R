function (r, nfactors = 1, n.obs = NA, n.iter = 1, rotate = "oblimin", 
    scores = "regression", residuals = FALSE, SMC = TRUE, covar = FALSE, 
    missing = FALSE, impute = "none", min.err = 0.001, max.iter = 50, 
    symmetric = TRUE, warnings = TRUE, fm = "minres", alpha = 0.1, 
    p = 0.05, oblique.scores = FALSE, np.obs = NULL, use = "pairwise", 
    cor = "cor", correct = 0.5, weight = NULL, n.rotations = 1, 
    hyper = 0.15, smooth = TRUE, ...) 
{
    cl <- match.call()
    if (isCorrelation(r)) {
        if (is.na(n.obs) && (n.iter > 1)) 
            stop("You must specify the number of subjects if giving a correlation matrix and doing confidence intervals")
        if (length(class(r)) > 1) {
            if (inherits(r, "partial.r")) 
                class(r) <- c("matrix", "array")
        }
    }
    f <- fac(r = r, nfactors = nfactors, n.obs = n.obs, rotate = rotate, 
        scores = scores, residuals = residuals, SMC = SMC, covar = covar, 
        missing = missing, impute = impute, min.err = min.err, 
        max.iter = max.iter, symmetric = symmetric, warnings = warnings, 
        fm = fm, alpha = alpha, oblique.scores = oblique.scores, 
        np.obs = np.obs, use = use, cor = cor, correct = correct, 
        weight = weight, n.rotations = n.rotations, hyper = hyper, 
        smooth = smooth, ... = ...)
    fl <- f$loadings
    nvar <- dim(fl)[1]
    if (n.iter > 1) {
        if (is.na(n.obs)) {
            n.obs <- f$n.obs
        }
        replicates <- list()
        rep.rots <- list()
        replicateslist <- parallel::mclapply(1:n.iter, function(x) {
            if (isCorrelation(r)) {
                mu <- rep(0, nvar)
                eX <- eigen(r)
                X <- matrix(rnorm(nvar * n.obs), n.obs)
                X <- t(eX$vectors %*% diag(sqrt(pmax(eX$values, 
                  0)), nvar) %*% t(X))
            }
            else {
                X <- r[sample(n.obs, n.obs, replace = TRUE), 
                  ]
            }
            fs <- fac(X, nfactors = nfactors, rotate = rotate, 
                scores = "none", SMC = SMC, missing = missing, 
                impute = impute, min.err = min.err, max.iter = max.iter, 
                symmetric = symmetric, warnings = warnings, fm = fm, 
                alpha = alpha, oblique.scores = oblique.scores, 
                np.obs = np.obs, use = use, cor = cor, correct = correct, 
                n.rotations = n.rotations, hyper = hyper, smooth = smooth, 
                ... = ...)
            if (nfactors == 1) {
                replicates <- list(loadings = fs$loadings)
            }
            else {
                t.rot <- target.rot(fs$loadings, fl)
                if (!is.null(fs$Phi)) {
                  phis <- fs$Phi
                  replicates <- list(loadings = t.rot$loadings, 
                    phis = phis[lower.tri(t.rot$Phi)])
                }
                else {
                  replicates <- list(loadings = t.rot$loadings)
                }
            }
        })
        replicates <- matrix(unlist(replicateslist), nrow = n.iter, 
            byrow = TRUE)
        means <- colMeans(replicates, na.rm = TRUE)
        sds <- apply(replicates, 2, sd, na.rm = TRUE)
        if (length(means) > (nvar * nfactors)) {
            means.rot <- means[(nvar * nfactors + 1):length(means)]
            sds.rot <- sds[(nvar * nfactors + 1):length(means)]
            ci.rot.lower <- means.rot + qnorm(p/2) * sds.rot
            ci.rot.upper <- means.rot + qnorm(1 - p/2) * sds.rot
            ci.rot <- data.frame(lower = ci.rot.lower, upper = ci.rot.upper)
        }
        else {
            rep.rots <- NULL
            means.rot <- NULL
            sds.rot <- NULL
            z.rot <- NULL
            ci.rot <- NULL
        }
        means <- matrix(means[1:(nvar * nfactors)], ncol = nfactors)
        sds <- matrix(sds[1:(nvar * nfactors)], ncol = nfactors)
        tci <- abs(means)/sds
        ptci <- 1 - pnorm(tci)
        if (!is.null(rep.rots)) {
            tcirot <- abs(means.rot)/sds.rot
            ptcirot <- 1 - pnorm(tcirot)
        }
        else {
            tcirot <- NULL
            ptcirot <- NULL
        }
        ci.lower <- means + qnorm(p/2) * sds
        ci.upper <- means + qnorm(1 - p/2) * sds
        ci <- data.frame(lower = ci.lower, upper = ci.upper)
        class(means) <- "loadings"
        colnames(means) <- colnames(sds) <- colnames(fl)
        rownames(means) <- rownames(sds) <- rownames(fl)
        f$cis <- list(means = means, sds = sds, ci = ci, p = 2 * 
            ptci, means.rot = means.rot, sds.rot = sds.rot, ci.rot = ci.rot, 
            p.rot = ptcirot, Call = cl, replicates = replicates, 
            rep.rots = rep.rots)
        results <- f
        results$Call <- cl
        class(results) <- c("psych", "fa.ci")
    }
    else {
        results <- f
        results$Call <- cl
        class(results) <- c("psych", "fa")
    }
    return(results)
}
<bytecode: 0x0000022a12295228>
<environment: namespace:psych>
