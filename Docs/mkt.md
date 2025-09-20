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
    CustomerIDCol string // The column index(A, B, C, ...) of customer ID in the data table
    TradingDayCol string // The column index(A, B, C, ...) of trading day in the data table
    AmountCol     string // The column index(A, B, C, ...) of amount in the data table
    NumGroups     uint   // The number of groups to divide the customers into
    DateFormat    string // The format of the date string (e.g., "YYYY-MM-DD", "DD/MM/YYYY", "yyyy-mm-dd")
}
```

**Fields:**

- `CustomerIDCol`: Column identifier for customer ID (e.g., "A", "B", "C")
- `TradingDayCol`: Column identifier for transaction date
- `AmountCol`: Column identifier for transaction amount
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
        CustomerIDCol: "A",
        TradingDayCol: "B",
        AmountCol:     "C",
        NumGroups:     5,  // 5-point scale
        DateFormat:    "2006-01-02",
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
// Different date formats
config := mkt.RFMConfig{
    CustomerIDCol: "CustomerID",
    TradingDayCol: "PurchaseDate",
    AmountCol:     "TotalAmount",
    NumGroups:     3,  // 3-point scale (Low/Medium/High)
    DateFormat:    "02/01/2006",  // DD/MM/YYYY format
}

// For European date format
config.DateFormat = "2006-01-02"  // ISO format
```

---

## Error Handling

The RFM function returns `nil` in the following cases:

- Invalid date format in transaction data
- Missing or invalid customer ID
- Date parsing errors

Always check the return value:

```go
result := mkt.RFM(dt, config)
if result == nil {
    // Handle error - check logs for details
    insyra.LogError("mkt", "RFM", "Analysis failed")
    return
}
```

---

## Best Practices

1. **Data Preparation**:
   - Ensure dates are in consistent format
   - Remove invalid or incomplete records
   - Verify customer IDs are unique per transaction

2. **Group Selection**:
   - Use 3-5 groups for most analyses
   - Higher group counts provide more granularity but may over-segment

3. **Date Format**:
   - Use Go time format strings (e.g., "2006-01-02" for YYYY-MM-DD)
   - Test with sample data first

4. **Performance**:
   - Function processes data in parallel for large datasets
   - Consider data size for very large transaction tables

---

## Related Packages

- [`insyra`](../README.md): Core data manipulation functions
- [`stats`](./stats.md): Statistical analysis functions
- [`isr`](./isr.md): Syntax sugar package for simplified usage
- [`parallel`](./parallel.md): Parallel computation utilities
