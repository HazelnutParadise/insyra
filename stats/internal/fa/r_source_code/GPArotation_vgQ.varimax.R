function (L) 
{
    QL <- sweep(L^2, 2, colMeans(L^2), "-")
    list(Gq = -L * QL, f = -sqrt(sum(diag(crossprod(QL))))^2/4, 
        Method = "varimax")
}
<bytecode: 0x0000022a11cf1120>
<environment: namespace:GPArotation>
