# [ mkt ] Package

This document describes all public APIs in the `mkt` package, designed for AI/automated applications to directly understand each function, type, parameter, and return value.

---

## Installation

```bash
go get github.com/HazelnutParadise/insyra/mkt
```

---

## Overview

The mkt package provides marketing analytics functions for customer segmentation and analysis:

- **RFM Analysis**: Customer segmentation based on Recency, Frequency, and Monetary value
- **CAI Analysis**: Customer Activity Index calculation to evaluate customer activity trends over time

---

## Core Types

### TimeScale

Time scale for RFM analysis recency calculation.

```go
type TimeScale string
```

**Constants:**

- `TimeScaleHourly`: Calculate recency in hours
- `TimeScaleDaily`: Calculate recency in days (default)
- `TimeScaleWeekly`: Calculate recency in weeks
- `TimeScaleMonthly`: Calculate recency in months
- `TimeScaleYearly`: Calculate recency in years

### RFMConfig

Configuration structure for RFM analysis.

```go
type RFMConfig struct {
    CustomerIDColIndex string    // The column index(A, B, C, ...) of customer ID in the data table
    CustomerIDColName  string    // The column name of customer ID in the data table (if both index and name are provided, index takes precedence)
    TradingDayColIndex string    // The column index(A, B, C, ...) of trading day in the data table
    TradingDayColName  string    // The column name of trading day in the data table (if both index and name are provided, index takes precedence)
    AmountColIndex     string    // The column index(A, B, C, ...) of amount in the data table
    AmountColName      string    // The column name of amount in the data table (if both index and name are provided, index takes precedence)
    NumGroups          uint      // The number of groups to divide the customers into
    DateFormat         string    // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
    TimeScale          TimeScale // The time scale for recency calculation (e.g., hourly, daily, weekly, monthly, yearly)
}
```

**Fields:**

- `CustomerIDColIndex`: Column index for customer ID (e.g., "A", "B", "C")
- `CustomerIDColName`: Column name for customer ID (column index takes precedence if both are provided)
- `TradingDayColIndex`: Column index for transaction date
- `TradingDayColName`: Column name for transaction date (column index takes precedence if both are provided)
- `AmountColIndex`: Column index for transaction amount
- `AmountColName`: Column name for transaction amount (column index takes precedence if both are provided)
- `NumGroups`: Number of RFM score groups (typically 3-5)
- `DateFormat`: Date format string using `YYYY`, `MM`, `DD` tokens (defaults to "YYYY-MM-DD" if empty)
- `TimeScale`: Time scale for recency calculation (defaults to "daily" if empty)

### CAIConfig

Configuration structure for CAI (Customer Activity Index) analysis.

```go
type CAIConfig struct {
    CustomerIDColIndex string    // The column index(A, B, C, ...) of customer ID in the data table
    CustomerIDColName  string    // The column name of customer ID in the data table (if both index and name are provided, index takes precedence)
    TradingDayColIndex string    // The column index(A, B, C, ...) of trading day in the data table
    TradingDayColName  string    // The column name of trading day in the data table (if both index and name are provided, index takes precedence)
    DateFormat         string    // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
    TimeScale          TimeScale // The time scale for analysis (e.g., hourly, daily, weekly, monthly, yearly)
}
```

**Fields:**

- `CustomerIDColIndex`: Column index for customer ID (e.g., "A", "B", "C")
- `CustomerIDColName`: Column name for customer ID (column index takes precedence if both are provided)
- `TradingDayColIndex`: Column index for transaction date
- `TradingDayColName`: Column name for transaction date (column index takes precedence if both are provided)
- `DateFormat`: Date format string using `YYYY`, `MM`, `DD` tokens (defaults to "YYYY-MM-DD" if empty)
- `TimeScale`: Time scale for analysis (defaults to "daily" if empty)

---

## Functions

### RFM

RFM analysis segments customers based on three key metrics:

- **Recency (R)**: Time since last purchase in the specified time scale (lower values = higher scores)
- **Frequency (F)**: Number of unique transaction periods based on TimeScale (higher values = higher scores). For example, if TimeScale is `daily`, multiple transactions on the same day count as one; if `weekly`, multiple transactions in the same week count as one.
- **Monetary (M)**: Total purchase amount (higher values = higher scores)

```go
func RFM(dt insyra.IDataTable, rfmConfig RFMConfig) insyra.IDataTable
```

**Description:**
Performs RFM (Recency, Frequency, Monetary) analysis on customer transaction data.

The function calculates percentile-based scores for each metric and assigns customers to groups. The TimeScale parameter determines how recency is calculated (hours, days, weeks, months, or years) and also affects how Frequency is counted (transactions within the same time period are counted as one).

**Parameters:**

- `dt`: Input data table containing customer transaction data
- `rfmConfig`: Configuration for RFM analysis

**Returns:**

- `insyra.IDataTable`: Result table with RFM scores, or `nil` if an error occurs

**Output Table Structure:**

- `CustomerID`: Customer identifier
- `R_Score`: Recency score (1 to NumGroups, higher is better)
- `F_Score`: Frequency score (1 to NumGroups, higher is better)
- `M_Score`: Monetary score (1 to NumGroups, higher is better)
- `RFM_Score`: The score calculated by R_Score, F_Score, and M_Score (default: sum of R, F, M scores).

### CAI

Alias for CustomerActivityIndex function.

```go
var CAI = CustomerActivityIndex
```

### Customer Activity Index

CAI (Customer Activity Index) is a metric used to evaluate customer activity based on their transaction history. It indicates the change in customer activity level over time. A positive CAI indicates a customer whose activity is increasing, while a negative CAI indicates a customer whose activity is decreasing.

The calculation involves:

- **MLE (Mean Lifetime Expectancy)**: Average time interval between transactions
- **WMLE (Weighted Mean Lifetime Expectancy)**: Weighted average where recent intervals have higher weights
- **CAI**: Calculated as (MLE - WMLE) / MLE

```go
func CustomerActivityIndex(dt insyra.IDataTable, caiConfig CAIConfig) insyra.IDataTable
```

**Description:**

Calculates the Customer Activity Index (CAI) for each customer based on their transaction history.

**Parameters:**

- `dt`: Input data table containing transaction records
- `caiConfig`: Configuration for CAI calculation

**Returns:**

- `insyra.IDataTable`: Result table with CustomerID, MLE, WMLE, and CAI for each customer, or `nil` if an error occurs

> [!NOTE]
> Only customers with at least 4 transactions are considered for CAI calculation.

**Output Table Structure:**

- `CustomerID`: Customer identifier
- `MLE`: Mean Lifetime Expectancy (average transaction interval)
- `WMLE`: Weighted Mean Lifetime Expectancy
- `CAI`: Customer Activity Index (activity trend indicator)

---

## Usage Examples

### Basic RFM Analysis

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/mkt"
)

func main() {
    // Create sample transaction data
    dt := insyra.NewDataTable()

    // Add transaction records
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C001",        // Customer ID
        "B": "2023-01-15",  // Transaction Date
        "C": 150.0,         // Amount
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C001",
        "B": "2023-02-20",
        "C": 200.0,
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C002",
        "B": "2023-03-10",
        "C": 300.0,
    })

    // Set column names
    dt.SetColNameByIndex("A", "CustomerID")
    dt.SetColNameByIndex("B", "Date")
    dt.SetColNameByIndex("C", "Amount")

    // Configure RFM analysis
    config := mkt.RFMConfig{
        CustomerIDColIndex: "A",
        TradingDayColIndex: "B",
        AmountColIndex:     "C",
        NumGroups:          5,  // 5-point scale
        DateFormat:         "2006-01-02",
    }

    // Perform RFM analysis
    result := mkt.RFM(dt, config)
    if result == nil {
        fmt.Println("RFM analysis failed")
        return
    }

    // Display results
    result.Show()

    // Access individual scores
    numRows, _ := result.Size()
    for i := 0; i < numRows; i++ {
        customerID := result.GetElement(i, "A")
        rfmScore := result.GetElement(i, "E")  // RFM_Score column
        fmt.Printf("Customer %s has RFM score: %s\n", customerID, rfmScore)
    }
}
```

### Advanced Configuration

```go
// Using column names instead of indices
config := mkt.RFMConfig{
    CustomerIDColName:  "CustomerID",
    TradingDayColName:  "PurchaseDate",
    AmountColName:      "TotalAmount",
    NumGroups:          3,  // 3-point scale (Low/Medium/High)
    DateFormat:         "02/01/2006",  // DD/MM/YYYY format
}

// For European date format
config.DateFormat = "2006-01-02"  // ISO format

// Mixed usage (index takes precedence if both are provided)
config := mkt.RFMConfig{
    CustomerIDColIndex: "A",           // This will be used
    CustomerIDColName:  "CustomerID",  // Ignored
    TradingDayColIndex: "B",
    AmountColIndex:     "C",
    NumGroups:          5,
    DateFormat:         "2006-01-02",
}
```

### RFM Analysis with TimeScale

```go
// Configure RFM analysis with weekly time scale
config := mkt.RFMConfig{
    CustomerIDColName:  "CustomerID",
    TradingDayColName:  "PurchaseDate",
    AmountColName:      "TotalAmount",
    NumGroups:          5,
    DateFormat:         "2006-01-02",
    TimeScale:          mkt.TimeScaleWeekly,  // Calculate recency in weeks
}

// Perform RFM analysis
result := mkt.RFM(dt, config)

// Recency scores will now be calculated in weeks instead of days
// For example, a customer who purchased 2 weeks ago will have R=2
```

### Basic CAI Analysis

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/mkt"
)

func main() {
    // Create sample transaction data
    dt := insyra.NewDataTable()

    // Add transaction records
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C001",        // Customer ID
        "B": "2023-01-01",  // Transaction Date
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C001",
        "B": "2023-01-15",
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C001",
        "B": "2023-02-01",
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C002",
        "B": "2023-01-10",
    })
    dt.AppendRowsByColIndex(map[string]any{
        "A": "C002",
        "B": "2023-02-15",
    })

    // Set column names
    dt.SetColNameByIndex("A", "CustomerID")
    dt.SetColNameByIndex("B", "Date")

    // Configure CAI analysis
    config := mkt.CAIConfig{
        CustomerIDColIndex: "A",
        TradingDayColIndex: "B",
        DateFormat:         "2006-01-02",
        TimeScale:          mkt.TimeScaleDaily,
    }

    // Perform CAI analysis
    result := mkt.CAI(dt, config)
    if result == nil {
        fmt.Println("CAI analysis failed")
        return
    }

    // Display results
    result.Show()

    // Access individual CAI values
    numRows, _ := result.Size()
    for i := 0; i < numRows; i++ {
        customerID := result.GetElement(i, "CustomerID")
        cai := result.GetElement(i, "CAI")
        mle := result.GetElement(i, "MLE")
        wmle := result.GetElement(i, "WMLE")
        fmt.Printf("Customer %s: CAI=%.3f, MLE=%.1f, WMLE=%.1f\n", customerID, cai, mle, wmle)
    }
}
```

### CAI Analysis with Different Time Scales

```go
// Configure CAI analysis with weekly time scale
config := mkt.CAIConfig{
    CustomerIDColName:  "CustomerID",
    TradingDayColName:  "PurchaseDate",
    DateFormat:         "2006-01-02",
    TimeScale:          mkt.TimeScaleWeekly,  // Calculate intervals in weeks
}

// Perform CAI analysis
result := mkt.CAI(dt, config)

// CAI values will be calculated based on weekly intervals
// Positive CAI indicates increasing activity, negative indicates decreasing
```

---

## Error Handling

The RFM function returns `nil` in the following cases:

- Invalid date format in transaction data
- Missing or invalid customer ID
- Date parsing errors
- Missing required configuration fields (CustomerID, TradingDay, Amount columns)

Always check the return value:

```go
result := mkt.RFM(dt, config)
if result == nil {
    // Handle error - check logs for details
    insyra.LogError("mkt", "RFM", "Analysis failed")
    return
}
```

The function logs warnings for specific issues:

- Invalid date strings
- Missing column specifications
- Data parsing failures

The CAI function returns `nil` in the following cases:

- Invalid date format in transaction data
- Missing or invalid customer ID
- Date parsing errors
- Missing required configuration fields (CustomerID, TradingDay columns)

Always check the return value:

```go
result := mkt.CAI(dt, config)
if result == nil {
    // Handle error - check logs for details
    insyra.LogError("mkt", "CAI", "Analysis failed")
    return
}
```

The function logs warnings for specific issues:

- Invalid date strings
- Missing column specifications
- Data parsing failures
- Customers with insufficient transaction data (less than 2 transactions)

---

## Best Practices

1. **Data Preparation**:

   - Ensure dates are in consistent format
   - Remove invalid or incomplete records
   - Verify customer IDs are unique per transaction

2. **Column Specification**:

   - Use column names for better readability (e.g., "CustomerID" vs "A")
   - Column indices take precedence if both name and index are provided
   - Ensure specified columns exist in the data table

3. **Group Selection**:

   - Use 3-5 groups for most analyses
   - Higher group counts provide more granularity but may over-segment

4. **Date Format**:

   - Use Go time format strings (e.g., "2006-01-02" for YYYY-MM-DD)
   - Test with sample data first

5. **Time Scale Selection**:

   - Choose TimeScale based on business context (daily for retail, weekly for B2B)
   - Default is daily if TimeScale is not specified
   - Recency calculation truncates times to the start of the time unit

6. **Performance**:

   - Function processes data in parallel for large datasets
   - Consider data size for very large transaction tables

7. **CAI Analysis Considerations**:
   - Ensure customers have sufficient transaction history (at least 2 transactions)
   - Choose appropriate TimeScale based on business cycle length
   - CAI values close to 0 indicate stable activity patterns
   - Positive CAI indicates increasing activity, negative indicates decreasing
   - Use CAI in combination with RFM for comprehensive customer insights

---

## Related Packages

- [`insyra`](../README.md): Core data manipulation functions
- [`stats`](./stats.md): Statistical analysis functions
- [`isr`](./isr.md): Syntax sugar package for simplified usage
- [`parallel`](./parallel.md): Parallel computation utilities
