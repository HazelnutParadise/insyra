function (A, Tmat = diag(ncol(A)), gam = 0, normalize = FALSE, 
    eps = 1e-05, maxit = 1000, randomStarts = 0) 
{
    GPFRSoblq(A, Tmat = Tmat, normalize = normalize, eps = eps, 
        maxit = maxit, method = "oblimin", methodArgs = list(gam = gam), 
        randomStarts = randomStarts)
}
<bytecode: 0x0000022a11f87898>
<environment: namespace:GPArotation>
