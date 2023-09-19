## Lecture 1 (Introduction)

[Lecture1.txt](https://pdos.csail.mit.edu/6.824/notes/l01.txt)

This class is mostly about infrastruction services.

- Benifit:

  - Increase capacity.

    Tolerate fault.

  - Match disribution of physical devices.

  - Increase security via isolation.

- Difficulty:

  - Concurrency
  - Complex interactions
  - Partial failure
  - Hard to get high performance

### Main topics

Three infrastructure for applications.

- Storage
- Communication
- Computation

#### Fault tolerance(Lab 2 and 3)

The bit network always have something broken, we'd like hide the failures. "High availability": server continues despite failures. **Replicated servers**, if one crashed, use another.

#### Consistency

Key-value question.

#### Performance(Lab 1 and 4)

The goal is to scalable throughput. The scaling gets harder in case of load imbalance, slowest of n latency, interaction.

#### Tradeoffs

Fault-tolerance, consistency, performance are enemies.

Fault tolerance and consistenct require communication.

Many designs provide only weak consistency to gain speed.

#### Implementation

​	RPC, threads, concurrency control, configuration.

### Case study : MapReduce

- Context: multi-hour computations on multi-terabyte data-sets.

- Goal: easy fo r non-specialist programmers.

  Programmer just defines Map and Reduce functions, the functions are just functional and stateless.

  Write sequential code and MR deal with all the distributedness.

- Abstract view

  ```
  	Input1 -> Map -> a,1 b,1
    Input2 -> Map ->     b,1
    Input3 -> Map -> a,1     c,1
                      |   |   |
                      |   |   -> Reduce -> c,1
                      |   -----> Reduce -> b,2
                      ---------> Reduce -> a,2
  ```

  ```
    Map(k, v)
      split v into words
      for each word w
        emit(w, "1")
    Reduce(k, v_list)
      emit(len(v_list))
  ```

### Fault tolerance

Coordinator reruns MR functions.

The map and reduce functions can run twice, it means the same input will generate the same output.

- Coordinator failure, rerun the function. The coordinator cannot fail.
- Slow workers(struggler), rerun it on different worker.

