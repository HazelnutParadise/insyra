function (fa.results, Phi = NULL, fe.results = NULL, sort = TRUE, 
    labels = NULL, cut = 0.3, simple = TRUE, errors = FALSE, 
    g = FALSE, digits = 1, e.size = 0.05, rsize = 0.15, side = 2, 
    main, cex = NULL, l.cex = NULL, marg = c(0.5, 0.5, 1, 0.5), 
    adj = 1, ic = FALSE, dictionary = NULL, ...) 
{
    if (length(class(fa.results)) > 1) {
        if (inherits(fa.results, "principal")) {
            pc <- TRUE
        }
        else {
            pc <- FALSE
        }
    }
    else {
        pc <- FALSE
    }
    if (ic) 
        pc <- TRUE
    old.par <- par(mar = marg)
    on.exit(par(old.par))
    col <- c("black", "red")
    if (missing(main)) 
        if (is.null(fe.results)) {
            if (pc) {
                main <- "Components Analysis"
            }
            else {
                main <- "Factor Analysis"
            }
        }
        else {
            main <- "Factor analysis and extension"
        }
    if (!is.matrix(fa.results) && !is.null(fa.results$fa) && 
        is.list(fa.results$fa)) 
        fa.results <- fa.results$fa
    if (is.null(cex)) 
        cex <- 1
    if (is.null(l.cex)) 
        l.cex <- 1
    if (!is.null(dictionary)) {
        if ("Content" %in% colnames(dictionary)) 
            vName = "Content"
        if ("item" %in% colnames(dictionary)) 
            vName = "item"
        if ("Item" %in% colnames(dictionary)) 
            vName = "Item"
    }
    if (sort) {
        if (g) {
            temp <- fa.sort(fa.results[, -1])
            temp2 <- fa.results[, 1]
            fa.results <- cbind(g = temp2[rownames(temp)], temp)
        }
        else {
            fa.results <- fa.sort(fa.results)
        }
        if (!is.null(fe.results)) {
            fe.results <- fa.sort(fe.results)
        }
    }
    if ((!is.matrix(fa.results)) && (!is.data.frame(fa.results))) {
        factors <- as.matrix(fa.results$loadings)
        if (!is.null(fa.results$Phi)) {
            Phi <- fa.results$Phi
        }
        else {
            if (!is.null(fa.results$cor)) {
                Phi <- fa.results$cor
            }
        }
    }
    else {
        factors <- fa.results
    }
    if (!is.null(dictionary)) 
        rownames(factors) <- dictionary[rownames(factors), vName]
    nvar <- dim(factors)[1]
    if (is.null(nvar)) {
        nvar <- length(factors)
        num.factors <- 1
    }
    else {
        num.factors <- dim(factors)[2]
    }
    nvar <- dim(factors)[1]
    e.size = e.size * 16 * cex/nvar
    if (is.null(nvar)) {
        nvar <- length(factors)
        num.factors <- 1
    }
    else {
        num.factors <- dim(factors)[2]
    }
    if (is.null(rownames(factors))) {
        rownames(factors) <- paste("V", 1:nvar, sep = "")
    }
    if (is.null(colnames(factors))) {
        colnames(factors) <- paste("F", 1:num.factors, sep = "")
    }
    var.rect <- list()
    fact.rect <- list()
    max.len <- max(nchar(rownames(factors))) * rsize
    x.max <- max((nvar + 1), 6)
    limx = c(-max.len/2, x.max)
    n.evar <- 0
    if (!is.null(fe.results)) {
        n.evar <- dim(fe.results$loadings)[1]
        limy <- c(0, max(nvar + 1, n.evar + 1))
    }
    else {
        limy = c(0, nvar + 1)
    }
    top <- max(nvar, n.evar) + 1
    plot(0, type = "n", xlim = limx, ylim = limy, frame.plot = FALSE, 
        axes = FALSE, ylab = "", xlab = "", main = main, ...)
    max.len <- max(strwidth(rownames(factors)), strwidth("abc"))/1.8
    limx = c(-max.len/2, x.max)
    cex <- min(cex, 20/x.max)
    if (g) {
        left <- 0.3 * x.max
        middle <- 0.6 * x.max
        gf <- 2
    }
    else {
        left <- 0
        middle <- 0.5 * x.max
        gf <- 1
    }
    for (v in 1:nvar) {
        var.rect[[v]] <- dia.rect(left, top - v - max(0, n.evar - 
            nvar)/2, rownames(factors)[v], xlim = limx, ylim = limy, 
            cex = cex, draw = FALSE, ...)
    }
    all.rects.x <- rep(left, nvar)
    all.rects.y <- top - 1:nvar - max(0, n.evar - nvar)/2
    all.rects.names <- rownames(factors)[1:nvar]
    dia.rect(all.rects.x, all.rects.y, all.rects.names)
    f.scale <- (top)/(num.factors + 1)
    f.shift <- max(nvar, n.evar)/num.factors
    if (g) {
        fact.rect[[1]] <- dia.ellipse(-max.len/2, top/2, colnames(factors)[1], 
            xlim = limx, ylim = limy, e.size = e.size, cex = cex, 
            ...)
        for (v in 1:nvar) {
            if (simple && (abs(factors[v, 1]) == max(abs(factors[v, 
                ]))) && (abs(factors[v, 1]) > cut) | (!simple && 
                (abs(factors[v, 1]) > cut))) {
                dia.arrow(from = fact.rect[[1]], to = var.rect[[v]]$left, 
                  labels = round(factors[v, 1], digits), col = ((sign(factors[v, 
                    1]) < 0) + 1), lty = ((sign(factors[v, 1]) < 
                    0) + 1))
            }
        }
    }
    text.values <- list()
    tv.index <- 1
    for (f in gf:num.factors) {
        if (pc) {
            fact.rect[[f]] <- dia.rect(left + middle, (num.factors + 
                gf - f) * f.scale, colnames(factors)[f], xlim = limx, 
                ylim = limy, cex = cex, draw = FALSE, ...)
        }
        else {
            fact.rect[[f]] <- dia.ellipse(left + middle, (num.factors + 
                gf - f) * f.scale, colnames(factors)[f], xlim = limx, 
                ylim = limy, e.size = e.size, cex = cex, draw = FALSE, 
                ...)
        }
        for (v in 1:nvar) {
            if (simple && (abs(factors[v, f]) == max(abs(factors[v, 
                ]))) && (abs(factors[v, f]) > cut) | (!simple && 
                (abs(factors[v, f]) > cut))) {
                if (pc) {
                  text.values[[tv.index]] <- dia.arrow(to = fact.rect[[f]], 
                    from = var.rect[[v]]$right, labels = round(factors[v, 
                      f], digits), col = ((sign(factors[v, f]) < 
                      0) + 1), lty = ((sign(factors[v, f]) < 
                      0) + 1), adj = f%%adj, cex = cex, draw = FALSE)
                  tv.index <- tv.index + 1
                }
                else {
                  text.values[[tv.index]] <- dia.arrow(from = fact.rect[[f]], 
                    to = var.rect[[v]]$right, labels = round(factors[v, 
                      f], digits), col = ((sign(factors[v, f]) < 
                      0) + 1), lty = ((sign(factors[v, f]) < 
                      0) + 1), adj = f%%adj + 1, cex = cex, draw = FALSE)
                  tv.index <- tv.index + 1
                }
            }
        }
    }
    tv <- matrix(unlist(fact.rect), nrow = num.factors, byrow = TRUE)
    all.rects.x <- tv[, 5]
    all.rects.y <- tv[, 2]
    all.rects.names <- colnames(factors)
    if (pc) {
        dia.rect(all.rects.x, all.rects.y, all.rects.names)
    }
    else {
        dia.multi.ellipse(all.rects.x, all.rects.y, all.rects.names, 
            e.size = e.size)
    }
    tv <- matrix(unlist(text.values), byrow = TRUE, ncol = 21)
    text(tv[, 1], tv[, 2], tv[, 3], cex = l.cex)
    arrows(x0 = tv[, 6], y0 = tv[, 7], x1 = tv[, 8], y1 = tv[, 
        9], length = tv[1, 10], angle = tv[1, 11], code = 1, 
        col = tv[, 20], lty = tv[, 21])
    arrows(x0 = tv[, 13], y0 = tv[, 14], x1 = tv[, 15], y1 = tv[, 
        16], length = tv[1, 17], angle = tv[1, 18], code = 2, 
        col = tv[, 20], lty = tv[, 21])
    if (!is.null(Phi) && (ncol(Phi) > 1)) {
        curve.list <- list()
        for (i in 2:num.factors) {
            for (j in 1:(i - 1)) {
                if (abs(Phi[i, j]) > cut) {
                  d.curve <- dia.curved.arrow(from = fact.rect[[j]]$right, 
                    to = fact.rect[[i]]$right, labels = round(Phi[i, 
                      j], digits), scale = (i - j), draw = FALSE, 
                    cex = cex, l.cex = l.cex, ...)
                  curve.list <- c(curve.list, d.curve)
                }
            }
        }
        multi.curved.arrow(curve.list, l.cex, ...)
    }
    self.list <- list()
    if (errors) {
        for (v in 1:nvar) {
            d.self <- dia.self(location = var.rect[[v]], scale = 0.5, 
                side = side)
            self.list <- c(self.list, d.self)
        }
    }
    if (length(self.list) > 0) 
        multi.self(self.list)
    if (!is.null(fe.results)) {
        e.loadings <- fe.results$loadings
        for (v in 1:n.evar) {
            var.rect[[v]] <- dia.rect(x.max, top - v - max(0, 
                nvar - n.evar)/2, rownames(e.loadings)[v], xlim = limx, 
                ylim = limy, cex = cex, ...)
            for (f in 1:num.factors) {
                if (simple && (abs(e.loadings[v, f]) == max(abs(e.loadings[v, 
                  ]))) && (abs(e.loadings[v, f]) > cut) | (!simple && 
                  (abs(e.loadings[v, f]) > cut))) {
                  dia.arrow(from = fact.rect[[f]], to = var.rect[[v]]$left, 
                    labels = round(e.loadings[v, f], digits), 
                    col = ((sign(e.loadings[v, f]) < 0) + 1), 
                    lty = ((sign(e.loadings[v, f]) < 0) + 1), 
                    adj = f%%adj + 1)
                }
            }
        }
    }
}
<bytecode: 0x0000022a131d66b0>
<environment: namespace:psych>
