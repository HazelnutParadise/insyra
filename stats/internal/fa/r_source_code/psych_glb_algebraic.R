# psych::glb.algebraic  (namespace: psych)
# dumped at 2025-10-04 13:16:30.682089
glb.algebraic <- 
function (Cov, LoBounds = NULL, UpBounds = NULL) 
{
    if (!requireNamespace("Rcsdp")) {
        stop("Rcsdp must be installed to find the glb.algebraic")
    }
    cl <- match.call()
    p <- dim(Cov)[2]
    if (dim(Cov)[1] != p) 
        Cov <- cov(Cov)
    if (any(t(Cov) != Cov)) 
        stop("'Cov' is not symmetric")
    if (is.null(LoBounds)) 
        LoBounds <- rep(0, ncol(Cov))
    if (is.null(UpBounds)) 
        UpBounds <- diag(Cov)
    if (any(LoBounds > UpBounds)) {
        stop("'LoBounds'<='UpBounds' violated")
    }
    if (length(LoBounds) != p) 
        stop("length(LoBounds) != dim(Cov)")
    if (length(UpBounds) != p) 
        stop("length(UpBounds)!=dim(Cov)")
    Var <- diag(Cov)
    opt = rep(1, p)
    C <- list(diag(Var) - Cov, -UpBounds, LoBounds)
    A <- vector("list", p)
    for (i in 1:p) {
        b <- rep(0, p)
        b[i] <- 1
        A[[i]] <- list(diag(b), -b, b)
    }
    K <- list(type = c("s", "l", "l"), size = rep(p, 3))
    result <- Rcsdp::csdp(C, A, opt, K, control = Rcsdp::csdp.control(printlevel = 0))
    if (result$status >= 4 || result$status == 2) {
        warning("Failure of csdp, status of solution=", result$status)
        lb <- list(glb = NA, solution = NA, status = result$status, 
            Call = cl)
    }
    else {
        if (result$status != 0) {
            warning("status of solution=", result$status)
        }
        item.diag <- result$y
        names(item.diag) <- colnames(Cov)
        lb <- list(glb = (sum(Cov) - sum(Var) + sum(result$y))/sum(Cov), 
            solution = item.diag, status = result$status, Call = cl)
    }
    return(lb)
}

