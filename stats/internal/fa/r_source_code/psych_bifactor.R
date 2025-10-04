function (L, Tmat = diag(ncol(L)), normalize = FALSE, eps = 1e-05, 
    maxit = 1000) 
{
    if (requireNamespace("GPArotation")) {
        GPArotation::GPForth(L, Tmat = Tmat, normalize = normalize, 
            eps = eps, maxit = maxit, method = "bimin")
    }
    else {
        stop("Bifactor requires GPArotation")
    }
}
<bytecode: 0x0000022a132c5018>
<environment: namespace:psych>
