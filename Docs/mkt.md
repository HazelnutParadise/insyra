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

---

## Core Types

### RFMConfig

Configuration structure for RFM analysis.

```go
type RFMConfig struct {
    CustomerIDColIndex string // The column index(A, B, C, ...) of customer ID in the data table
    CustomerIDColName  string // The column name of customer ID in the data table (if both index and name are provided, index takes precedence)
    TradingDayColIndex string // The column index(A, B, C, ...) of trading day in the data table
    TradingDayColName  string // The column name of trading day in the data table (if both index and name are provided, index takes precedence)
    AmountColIndex     string // The column index(A, B, C, ...) of amount in the data table
    AmountColName      string // The column name of amount in the data table (if both index and name are provided, index takes precedence)
    NumGroups          uint   // The number of groups to divide the customers into
    DateFormat         string // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
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
- `DateFormat`: Date format string (defaults to "YYYY-MM-DD" if empty)

---

## Functions

### RFM

Performs RFM (Recency, Frequency, Monetary) analysis on customer transaction data.

```go
func RFM(dt insyra.IDataTable, rfmConfig RFMConfig) insyra.IDataTable
```

**Parameters:**

- `dt`: Input data table containing customer transaction data
- `rfmConfig`: Configuration for RFM analysis

**Returns:**

- `insyra.IDataTable`: Result table with RFM scores, or `nil` if an error occurs

**Description:**
RFM analysis segments customers based on three key metrics:

- **Recency (R)**: Days since last purchase (lower values = higher scores)
- **Frequency (F)**: Number of purchases (higher values = higher scores)
- **Monetary (M)**: Total purchase amount (higher values = higher scores)

The function calculates percentile-based scores for each metric and assigns customers to groups.

**Output Table Structure:**

- `CustomerID`: Customer identifier
- `R_Score`: Recency score (1 to NumGroups, higher is better)
- `F_Score`: Frequency score (1 to NumGroups, higher is better)
- `M_Score`: Monetary score (1 to NumGroups, higher is better)
- `RFM_Score`: Combined RFM score (e.g., "555", "123")

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

5. **Performance**:
   - Function processes data in parallel for large datasets
   - Consider data size for very large transaction tables

---

## Related Packages

- [`insyra`](../README.md): Core data manipulation functions
- [`stats`](./stats.md): Statistical analysis functions
- [`isr`](./isr.md): Syntax sugar package for simplified usage
- [`parallel`](./parallel.md): Parallel computation utilities
