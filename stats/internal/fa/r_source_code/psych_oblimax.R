function (A, Tmat = diag(ncol(A)), normalize = FALSE, eps = 1e-05, 
    maxit = 1000, randomStarts = 0) 
{
    GPFRSoblq(A, Tmat = Tmat, normalize = normalize, eps = eps, 
        maxit = maxit, method = "oblimax", methodArgs = NULL, 
        randomStarts = randomStarts)
}
<bytecode: 0x0000022a1216d7e0>
<environment: namespace:GPArotation>
