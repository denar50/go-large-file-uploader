# Problem statement

Upload a large file to a server:
Calculate the checksum of the entire file and call the backend to initiate the transfer. 

Run 10 routines. Each routine will request the next chunk to send. Read the chunk from the file system, calculate the checksum, send it to the server. Send the file, read the next chunk until there are no chunks to read. Exit.
Retry 3 times and fail the whole transfer if the retries fail.

Server, allocate a temporal file of that big size in disk. Every time it receives a chunk, it should validate its checksum and write the section in the temporal file.

# API

`POST /upload`
{
    size: int64
    checksum: string
    name: string
}
response
{
    id: string
    chunk_size: int
    chunk_count: int
}
`PUT /upload/{id}/chunk`
{
    checksum: string
    from_position: int64
    chunk_number: int
}

`GET /upload/{id}`
response
{
    checksum: string
    chunk_count: int
}

`GET /upload/{id}/{chunk_number}`
response
{
    checksum: string
}
[CHUNK]