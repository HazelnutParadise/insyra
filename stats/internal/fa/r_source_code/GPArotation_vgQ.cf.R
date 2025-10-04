# GPArotation::vgQ.cf  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.692094
vgQ.cf <- 
function (L, kappa = 0) 
{
    k <- ncol(L)
    p <- nrow(L)
    N <- matrix(1, k, k) - diag(k)
    M <- matrix(1, p, p) - diag(p)
    L2 <- L^2
    f1 <- (1 - kappa) * sum(diag(crossprod(L2, L2 %*% N)))/4
    f2 <- kappa * sum(diag(crossprod(L2, M %*% L2)))/4
    list(Gq = (1 - kappa) * L * (L2 %*% N) + kappa * L * (M %*% 
        L2), f = f1 + f2, Method = if (kappa == 0) "Crawford-Ferguson Quartimax/Quartimin" else if (kappa == 
        1/p) "Crawford-Ferguson Varimax" else if (kappa == k/(2 * 
        p)) "Equamax" else if (kappa == (k - 1)/(p + k - 2)) "Parsimax" else if (kappa == 
        1) "Factor Parsimony" else paste("Crawford-Ferguson:k=", 
        kappa, sep = ""))
}

