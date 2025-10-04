function (L, Tmat = diag(ncol(L)), eps = 1e-05, maxit = 1000) 
{
    kappa = ncol(L)/(2 * nrow(L))
    if (requireNamespace("GPArotation")) {
        GPArotation::cfT(L, Tmat = diag(ncol(L)), kappa = kappa, 
            eps = eps, maxit = maxit)
    }
    else {
        stop("biquartimin requires GPArotation")
    }
}
<bytecode: 0x0000022a1224c638>
<environment: namespace:psych>
