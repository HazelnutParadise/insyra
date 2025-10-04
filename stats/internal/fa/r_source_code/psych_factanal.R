function (x, factors, data = NULL, covmat = NULL, n.obs = NA, 
    subset, na.action, start = NULL, scores = c("none", "regression", 
        "Bartlett"), rotation = "varimax", control = NULL, ...) 
{
    sortLoadings <- function(Lambda, rotm = NULL) {
        cn <- colnames(Lambda)
        Phi <- attr(Lambda, "covariance")
        io.ssq <- order(-if (is.null(Phi)) 
            colSums(Lambda^2)
        else diag(Phi %*% crossprod(Lambda)))
        Lambda <- Lambda[, io.ssq, drop = FALSE]
        neg <- colSums(Lambda) < 0
        Lambda[, neg] <- -Lambda[, neg]
        colnames(Lambda) <- cn
        has.rot <- !is.null(rotm)
        has.Ph <- !is.null(Phi)
        if (has.rot || has.Ph) 
            unit <- c(1, -1)[neg + 1L]
        if (has.rot) 
            rotm <- rotm[, io.ssq] %*% diag(unit)
        if (has.Ph) {
            attr(Lambda, "covariance") <- diag(unit) %*% Phi[io.ssq, 
                io.ssq] %*% diag(unit)
        }
        list(Lambda = Lambda, rotm = rotm)
    }
    cl <- match.call()
    na.act <- NULL
    if (is.list(covmat)) {
        if (anyNA(match(c("cov", "n.obs"), names(covmat)))) 
            stop("'covmat' is not a valid covariance list")
        cv <- covmat$cov
        n.obs <- covmat$n.obs
        have.x <- FALSE
    }
    else if (is.matrix(covmat)) {
        cv <- covmat
        have.x <- FALSE
    }
    else if (is.null(covmat)) {
        if (missing(x)) 
            stop("neither 'x' nor 'covmat' supplied")
        have.x <- TRUE
        if (inherits(x, "formula")) {
            mt <- terms(x, data = data)
            if (attr(mt, "response") > 0) 
                stop("response not allowed in formula")
            attr(mt, "intercept") <- 0
            mf <- match.call(expand.dots = FALSE)
            names(mf)[names(mf) == "x"] <- "formula"
            mf$factors <- mf$covmat <- mf$scores <- mf$start <- mf$rotation <- mf$control <- mf$... <- NULL
            mf[[1L]] <- quote(stats::model.frame)
            mf <- eval.parent(mf)
            na.act <- attr(mf, "na.action")
            if (.check_vars_numeric(mf)) 
                stop("factor analysis applies only to numerical variables")
            z <- model.matrix(mt, mf)
        }
        else {
            z <- as.matrix(x)
            if (!is.numeric(z)) 
                stop("factor analysis applies only to numerical variables")
            if (!missing(subset)) 
                z <- z[subset, , drop = FALSE]
        }
        covmat <- cov.wt(z)
        cv <- covmat$cov
        n.obs <- covmat$n.obs
    }
    else stop("'covmat' is of unknown type")
    scores <- match.arg(scores)
    if (scores != "none" && !have.x) 
        stop("requested scores without an 'x' matrix")
    p <- ncol(cv)
    if (p < 3) 
        stop("factor analysis requires at least three variables")
    dof <- 0.5 * ((p - factors)^2 - p - factors)
    if (dof < 0) 
        stop(sprintf(ngettext(factors, "%d factor is too many for %d variables", 
            "%d factors are too many for %d variables"), factors, 
            p), domain = NA)
    sds <- sqrt(diag(cv))
    cv <- cv/(sds %o% sds)
    cn <- list(nstart = 1, trace = FALSE, lower = 0.005)
    cn[names(control)] <- control
    more <- if (...length()) 
        Filter(length, list(...)[c("nstart", "trace", "lower", 
            "opt", "rotate")])
    if (length(more)) 
        cn[names(more)] <- more
    if (is.null(start)) {
        start <- (1 - 0.5 * factors/p)/diag(solve(cv))
        if ((ns <- cn$nstart) > 1) 
            start <- cbind(start, matrix(runif(ns - 1), p, ns - 
                1, byrow = TRUE))
    }
    start <- as.matrix(start)
    if (nrow(start) != p) 
        stop(sprintf(ngettext(p, "'start' must have %d row", 
            "'start' must have %d rows"), p), domain = NA)
    nc <- ncol(start)
    if (nc < 1) 
        stop("no starting values supplied")
    best <- Inf
    for (i in 1L:nc) {
        nfit <- factanal.fit.mle(cv, factors, start[, i], max(cn$lower, 
            0), cn$opt)
        if (cn$trace) 
            cat("start", i, "value:", format(nfit$criteria[1L]), 
                "uniqs:", format(as.vector(round(nfit$uniquenesses, 
                  4))), "\n")
        if (nfit$converged && nfit$criteria[1L] < best) {
            fit <- nfit
            best <- fit$criteria[1L]
        }
    }
    if (best == Inf) 
        stop(ngettext(nc, "unable to optimize from this starting value", 
            "unable to optimize from these starting values"), 
            domain = NA)
    load <- sortLoadings(fit$loadings)$Lambda
    if (is.function(rotation) || rotation != "none") {
        rot <- do.call(rotation, c(list(load), cn$rotate))
        load <- if (is.list(rot)) {
            fit$rotmat <- if (inherits(rot, "GPArotation")) 
                t(solve(rot$Th))
            else rot$rotmat
            rot$loadings
        }
        else rot
        if (is.list(rot) && !is.null(rot$Phi)) 
            attr(load, "covariance") <- rot$Phi
    }
    loadrot <- sortLoadings(load, fit$rotmat)
    fit$loadings <- loadrot$Lambda
    if (!is.null(loadrot$rotm)) 
        fit$rotmat <- loadrot$rotm
    class(fit$loadings) <- "loadings"
    fit$na.action <- na.act
    if (have.x && scores != "none") {
        Lambda <- fit$loadings
        zz <- scale(z, TRUE, TRUE)
        switch(scores, regression = {
            sc <- zz %*% solve(cv, Lambda)
            if (!is.null(Phi <- attr(Lambda, "covariance"))) sc <- sc %*% 
                Phi
        }, Bartlett = {
            d <- 1/fit$uniquenesses
            tmp <- t(Lambda * d)
            sc <- t(solve(tmp %*% Lambda, tmp %*% t(zz)))
        })
        rownames(sc) <- rownames(z)
        colnames(sc) <- colnames(Lambda)
        if (!is.null(na.act)) 
            sc <- napredict(na.act, sc)
        fit$scores <- sc
    }
    if (!is.na(n.obs) && dof > 0) {
        fit$STATISTIC <- (n.obs - 1 - (2 * p + 5)/6 - (2 * factors)/3) * 
            fit$criteria["objective"]
        fit$PVAL <- pchisq(fit$STATISTIC, dof, lower.tail = FALSE)
    }
    fit$n.obs <- n.obs
    fit$call <- cl
    fit
}
<bytecode: 0x0000022a10efaa20>
<environment: namespace:stats>
