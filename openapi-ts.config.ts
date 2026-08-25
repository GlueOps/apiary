export default {
    input: "./docs/openapi.json",
    output: "ui/src/sdk",
    plugins: [
        '@hey-api/client-fetch',
        '@hey-api/sdk',
        {
            name: '@tanstack/react-query',
            queryKeys: true,
            queryOptions: true,
            infiniteQueryKeys: true,
            infiniteQueryOptions: true,
            mutationOptions: true,
        },
    ],
};
