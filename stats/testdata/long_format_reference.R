## Reference data from R's own aov() and friedman.test() on long-format data
## whose rows were shuffled with a fixed seed.
## The R version that produced the numbers is the first line of the output (r.version).
## Regenerate: cd stats/testdata && Rscript long_format_reference.R > long_format_reference.txt

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  format(x, digits = 17, trim = TRUE)
}

emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")
emits <- function(key, s) cat(key, "=", s, "\n", sep = "")

emits("r.version", R.version.string)

# Case "tw" (two-way, balanced, text levels)
A_tw <- c()
B_tw <- c()
value_tw <- c(5.1, 4.8, 5.5, 6.2, 6.0, 6.9, 7.4, 7.9, 7.1, 4.2, 4.9, 4.4, 5.3, 5.8, 5.0, 6.1, 6.6, 6.8)
for (a in c("keto", "vegan")) {
  for (b in c("none", "walk", "run")) {
    for (rep in 1:3) {
      A_tw <- c(A_tw, a)
      B_tw <- c(B_tw, b)
    }
  }
}
set.seed(4301)
idx_tw <- sample(18)
value_tw_shuf <- value_tw[idx_tw]
A_tw_shuf <- A_tw[idx_tw]
B_tw_shuf <- B_tw[idx_tw]

emits("tw.value", paste(format(value_tw_shuf, digits = 17, trim = TRUE), collapse = ","))
emits("tw.A", paste(A_tw_shuf, collapse = ","))
emits("tw.B", paste(B_tw_shuf, collapse = ","))

fit_tw <- aov(value ~ A * B, data = data.frame(value = value_tw_shuf, A = factor(A_tw_shuf), B = factor(B_tw_shuf)))
s_tw <- summary(fit_tw)[[1]]
emit("tw.SSA", s_tw$`Sum Sq`[1])
emit("tw.SSB", s_tw$`Sum Sq`[2])
emit("tw.SSAB", s_tw$`Sum Sq`[3])
emit("tw.SSW", s_tw$`Sum Sq`[4])
emit("tw.DFA", s_tw$`Df`[1])
emit("tw.DFB", s_tw$`Df`[2])
emit("tw.DFAB", s_tw$`Df`[3])
emit("tw.DFW", s_tw$`Df`[4])
emit("tw.FA", s_tw$`F value`[1])
emit("tw.FB", s_tw$`F value`[2])
emit("tw.FAB", s_tw$`F value`[3])
emit("tw.PA", s_tw$`Pr(>F)`[1])
emit("tw.PB", s_tw$`Pr(>F)`[2])
emit("tw.PAB", s_tw$`Pr(>F)`[3])

# Case "tw_int" (two-way, balanced, INTEGER levels)
A_tw_int <- c()
B_tw_int <- c()
value_tw_int <- c(10, 12, 14, 15, 13, 11, 20, 22, 25, 27, 21, 24)
for (a in 1:2) {
  for (b in 1:3) {
    for (rep in 1:2) {
      A_tw_int <- c(A_tw_int, a)
      B_tw_int <- c(B_tw_int, b)
    }
  }
}
set.seed(4302)
idx_tw_int <- sample(12)
value_tw_int_shuf <- value_tw_int[idx_tw_int]
A_tw_int_shuf <- A_tw_int[idx_tw_int]
B_tw_int_shuf <- B_tw_int[idx_tw_int]

emits("tw_int.value", paste(format(value_tw_int_shuf, digits = 17, trim = TRUE), collapse = ","))
emits("tw_int.A", paste(A_tw_int_shuf, collapse = ","))
emits("tw_int.B", paste(B_tw_int_shuf, collapse = ","))

fit_tw_int <- aov(value ~ A * B, data = data.frame(value = value_tw_int_shuf, A = factor(A_tw_int_shuf), B = factor(B_tw_int_shuf)))
s_tw_int <- summary(fit_tw_int)[[1]]
emit("tw_int.SSA", s_tw_int$`Sum Sq`[1])
emit("tw_int.SSB", s_tw_int$`Sum Sq`[2])
emit("tw_int.SSAB", s_tw_int$`Sum Sq`[3])
emit("tw_int.SSW", s_tw_int$`Sum Sq`[4])
emit("tw_int.DFA", s_tw_int$`Df`[1])
emit("tw_int.DFB", s_tw_int$`Df`[2])
emit("tw_int.DFAB", s_tw_int$`Df`[3])
emit("tw_int.DFW", s_tw_int$`Df`[4])
emit("tw_int.FA", s_tw_int$`F value`[1])
emit("tw_int.FB", s_tw_int$`F value`[2])
emit("tw_int.FAB", s_tw_int$`F value`[3])
emit("tw_int.PA", s_tw_int$`Pr(>F)`[1])
emit("tw_int.PB", s_tw_int$`Pr(>F)`[2])
emit("tw_int.PAB", s_tw_int$`Pr(>F)`[3])

# Case "rm" (repeated measures)
subj_rm <- c()
cond_rm <- c()
value_rm <- c(10, 12, 15, 14, 11, 13, 13, 16, 9, 12, 14, 13, 12, 15, 17, 16, 10, 11, 14, 15, 13, 14, 16, 18)
for (s in paste0("s", 1:6)) {
  for (c in c("t1", "t2", "t3", "t4")) {
    subj_rm <- c(subj_rm, s)
    cond_rm <- c(cond_rm, c)
  }
}
set.seed(4303)
idx_rm <- sample(24)
value_rm_shuf <- value_rm[idx_rm]
subj_rm_shuf <- subj_rm[idx_rm]
cond_rm_shuf <- cond_rm[idx_rm]

emits("rm.value", paste(format(value_rm_shuf, digits = 17, trim = TRUE), collapse = ","))
emits("rm.cond", paste(cond_rm_shuf, collapse = ","))
emits("rm.subj", paste(subj_rm_shuf, collapse = ","))

df_rm <- data.frame(value = value_rm_shuf, cond = factor(cond_rm_shuf), subj = factor(subj_rm_shuf))
s_rm <- summary(aov(value ~ cond + Error(subj/cond), data = df_rm))

within_rm <- s_rm[["Error: subj:cond"]][[1]]
between_rm <- s_rm[["Error: subj"]][[1]]
if (is.null(within_rm)) stop("within (Error: subj:cond) is NULL")
if (is.null(between_rm)) stop("between (Error: subj) is NULL")

rm.SSB <- within_rm$`Sum Sq`[1]
rm.DFB <- within_rm$`Df`[1]
rm.F <- within_rm$`F value`[1]
rm.P <- within_rm$`Pr(>F)`[1]
rm.SSW <- within_rm$`Sum Sq`[2]
rm.DFW <- within_rm$`Df`[2]
rm.SSS <- between_rm$`Sum Sq`[1]
rm.DFS <- between_rm$`Df`[1]
rm.totalSS <- sum((value_rm_shuf - mean(value_rm_shuf))^2)
emit("rm.SSB", rm.SSB)
emit("rm.DFB", rm.DFB)
emit("rm.F", rm.F)
emit("rm.P", rm.P)
emit("rm.SSW", rm.SSW)
emit("rm.DFW", rm.DFW)
emit("rm.SSS", rm.SSS)
emit("rm.DFS", rm.DFS)
emit("rm.totalSS", rm.totalSS)
emit("rm.eta", rm.SSB / rm.totalSS)

# Case "fr" (Friedman with ties)
subj_fr <- c()
cond_fr <- c()
value_fr <- c(3, 5, 4, 2, 4, 4, 3, 3, 5, 1, 4, 3, 2, 5, 5, 3, 4, 2, 2, 3, 4)
for (s in paste0("p", 1:7)) {
  for (c in c("A", "B", "C")) {
    subj_fr <- c(subj_fr, s)
    cond_fr <- c(cond_fr, c)
  }
}
set.seed(4304)
idx_fr <- sample(21)
value_fr_shuf <- value_fr[idx_fr]
subj_fr_shuf <- subj_fr[idx_fr]
cond_fr_shuf <- cond_fr[idx_fr]

emits("fr.value", paste(format(value_fr_shuf, digits = 17, trim = TRUE), collapse = ","))
emits("fr.cond", paste(cond_fr_shuf, collapse = ","))
emits("fr.subj", paste(subj_fr_shuf, collapse = ","))

r_fr <- friedman.test(value ~ cond | subj, data = data.frame(value = value_fr_shuf, cond = factor(cond_fr_shuf), subj = factor(subj_fr_shuf)))
emit("fr.stat", unname(r_fr$statistic))
emit("fr.df", unname(r_fr$parameter))
emit("fr.p", r_fr$p.value)
emit("fr.W", r_fr$statistic / (7 * (3 - 1)))