# Sequence Diagram

```mermaid
sequenceDiagram
    main.go->>nonodo.go: NewSupervisor
    nonodo.go->>rollup: Start
    nonodo.go->>inspect: Start
    nonodo.go-->>main.go: ok
```
