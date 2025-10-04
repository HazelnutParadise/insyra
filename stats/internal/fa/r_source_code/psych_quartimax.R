function (A, Tmat = diag(ncol(A)), normalize = FALSE, eps = 1e-05, 
    maxit = 1000, randomStarts = 0) 
{
    GPFRSorth(A, Tmat = Tmat, method = "quartimax", normalize = normalize, 
        eps = eps, maxit = maxit, methodArgs = NULL, randomStarts = randomStarts)
}
<bytecode: 0x0000022a11f52370>
<environment: namespace:GPArotation>
