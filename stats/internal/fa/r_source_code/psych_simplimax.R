function (A, Tmat = diag(ncol(A)), k = nrow(A), normalize = FALSE, 
    eps = 1e-05, maxit = 1000, randomStarts = 0) 
{
    GPFRSoblq(A, Tmat = Tmat, normalize = normalize, eps = eps, 
        maxit = maxit, method = "simplimax", methodArgs = list(k = k), 
        randomStarts = randomStarts)
}
<bytecode: 0x0000022a127b0cf8>
<environment: namespace:GPArotation>
