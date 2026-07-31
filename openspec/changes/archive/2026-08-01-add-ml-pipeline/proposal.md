# Change: Compose preprocessing and a model into one fitted object

## Why
The protocol makes every model answer the same verbs. That is only half of what makes scikit-learn usable; the other half is that preprocessing and the model fit and predict as one thing.

Without it, a caller who scales, encodes and then fits has to remember to apply the same scaling and the same encoding, fitted on the training data, to every table they later score. Doing that by hand is where leakage comes from: fitting the scaler on all the data before splitting is the single most common mistake in applied machine learning, and it is invisible because the numbers look better, not worse.

A pipeline makes the correct thing the easy thing. It also makes cross-validation honest, since refitting a pipeline per fold refits its preprocessing per fold, which is the only way a fold's score means anything.

## What Changes
- Add a pipeline of transformer steps ending in a model, fitting as one and predicting as one
- Make a fitted pipeline satisfy `Model`, so anything that accepts a model accepts a pipeline
- Add a way to apply a transformer to named columns only, so numeric and categorical columns can be treated differently in one pipeline
- Report which step failed when fitting fails, since a pipeline error with no step name is nearly useless

## Impact
- Affected specs: `ml-pipeline`
- Affected code: `ml/`
- Depends on `add-ml-estimator-protocol`
