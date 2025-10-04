function (A, Tmat = diag(ncol(A)), normalize = FALSE, eps = 1e-05, 
    maxit = 1000, method = "quartimin", methodArgs = NULL) 
{
    if (1 >= ncol(A)) 
        stop("rotation does not make sense for single factor models.")
    if ((!is.logical(normalize)) || normalize) {
        W <- NormalizingWeight(A, normalize = normalize)
        normalize <- TRUE
        A <- A/W
    }
    al <- 1
    L <- A %*% t(solve(Tmat))
    Method <- paste("vgQ", method, sep = ".")
    VgQ <- do.call(Method, append(list(L), methodArgs))
    G <- -t(t(L) %*% VgQ$Gq %*% solve(Tmat))
    f <- VgQ$f
    Table <- NULL
    VgQt <- do.call(Method, append(list(L), methodArgs))
    for (iter in 0:maxit) {
        Gp <- G - Tmat %*% diag(c(rep(1, nrow(G)) %*% (Tmat * 
            G)))
        s <- sqrt(sum(diag(crossprod(Gp))))
        Table <- rbind(Table, c(iter, f, log10(s), al))
        if (s < eps) 
            break
        al <- 2 * al
        for (i in 0:10) {
            X <- Tmat - al * Gp
            v <- 1/sqrt(c(rep(1, nrow(X)) %*% X^2))
            Tmatt <- X %*% diag(v)
            L <- A %*% t(solve(Tmatt))
            VgQt <- do.call(Method, append(list(L), methodArgs))
            improvement <- f - VgQt$f
            if (improvement > 0.5 * s^2 * al) 
                break
            al <- al/2
        }
        Tmat <- Tmatt
        f <- VgQt$f
        G <- -t(t(L) %*% VgQt$Gq %*% solve(Tmatt))
    }
    convergence <- (s < eps)
    if ((iter == maxit) & !convergence) 
        warning("convergence not obtained in GPFoblq. ", maxit, 
            " iterations used.")
    if (normalize) 
        L <- L * W
    dimnames(L) <- dimnames(A)
    r <- list(loadings = L, Phi = t(Tmat) %*% Tmat, Th = Tmat, 
        Table = Table, method = VgQ$Method, orthogonal = FALSE, 
        convergence = convergence, Gq = VgQt$Gq)
    class(r) <- "GPArotation"
    r
}
<bytecode: 0x0000022a12391f90>
<environment: namespace:GPArotation>
