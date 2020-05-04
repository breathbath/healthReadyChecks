# Security Compliance

As part of security compliance documentation, this file lists all the data being either **stored** (e.g. DB, logs, etc) by “REPO_NAME” or **sent** (e.g. API calls, message bus, logs, etc) to other systems external to “SYSTEM_NAME”.


### How to fill document:
The goal of this document is to have an overview of which data is being processed by a component. This way security and privacy related issues can be quickly identified. With that in mind, here are some few pointers to help you out:

 - It is okay for data column to have a grouping or a set of data items lumped together that are suitable to be grouped, for example "device configuration" or "service info.", when necessary write in the notes column any clarifying details.
 -  Data column must be as granular as possible when it involves security or privacy critical data (e.g. passwords, customer personal info., certificates, LI etc.)
 - Whenever there is documentation like API definition already existing, please include those in the table.
 - Logs that are directly transmitted to a log collector entity via syslog for example are to be classified in "Data transmitted".
 - Target/location columns should easily identify the components listed and should not rely on network or host specific config. like IP address but on architecture level.
 - Use the "Security mechanism" column to highlight any security measure taken to protect the data during storage or transmission.
 - Try to be as thorough as possible with your listing, but we understand that it takes time to create a complete coverage of all data. So prioritize major or security related items.

## Data transmitted

| Target       | Protocol | Data                                                                                    | Security mechanism                   | Notes | Link to Documentation*                                                           |
|--------------|----------|-----------------------------------------------------------------------------------------|--------------------------------------|-------|----------------------------------------------------------------------------------|

## Message transmitted

|Massage     |Data         |Notes        |
|-------------|-------------|-------------|

## Data Stored

| Location | Data                                                      | Security mechanism | Notes                                                                                                                                                                                                                                                             |
|----------|-----------------------------------------------------------|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
