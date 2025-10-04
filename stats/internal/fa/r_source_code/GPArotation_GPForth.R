# GPArotation::GPForth  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.68379
GPForth <- 
function (A, Tmat = diag(ncol(A)), normalize = FALSE, eps = 1e-05, 
    maxit = 1000, method = "varimax", methodArgs = NULL) 
{
    if ((!is.logical(normalize)) || normalize) {
        W <- NormalizingWeight(A, normalize = normalize)
        normalize <- TRUE
        A <- A/W
    }
    if (1 >= ncol(A)) 
        stop("rotation does not make sense for single factor models.")
    al <- 1
    L <- A %*% Tmat
    Method <- paste("vgQ", method, sep = ".")
    VgQ <- do.call(Method, append(list(L), methodArgs))
    G <- crossprod(A, VgQ$Gq)
    f <- VgQ$f
    Table <- NULL
    VgQt <- do.call(Method, append(list(L), methodArgs))
    for (iter in 0:maxit) {
        M <- crossprod(Tmat, G)
        S <- (M + t(M))/2
        Gp <- G - Tmat %*% S
        s <- sqrt(sum(diag(crossprod(Gp))))
        Table <- rbind(Table, c(iter, f, log10(s), al))
        if (s < eps) 
            break
        al <- 2 * al
        for (i in 0:10) {
            X <- Tmat - al * Gp
            UDV <- svd(X)
            Tmatt <- UDV$u %*% t(UDV$v)
            L <- A %*% Tmatt
            VgQt <- do.call(Method, append(list(L), methodArgs))
            if (VgQt$f < (f - 0.5 * s^2 * al)) 
                break
            al <- al/2
        }
        Tmat <- Tmatt
        f <- VgQt$f
        G <- crossprod(A, VgQt$Gq)
    }
    convergence <- (s < eps)
    if ((iter == maxit) & !convergence) 
        warning("convergence not obtained in GPForth. ", maxit, 
            " iterations used.")
    if (normalize) 
        L <- L * W
    dimnames(L) <- dimnames(A)
    r <- list(loadings = L, Th = Tmat, Table = Table, method = VgQ$Method, 
        orthogonal = TRUE, convergence = convergence, Gq = VgQt$Gq)
    class(r) <- "GPArotation"
    r
}

