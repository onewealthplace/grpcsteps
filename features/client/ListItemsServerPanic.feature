Feature: List Items with server panic

    Scenario: Server Panic
        When I request a gRPC method "/grpctest.ItemService/ListItems" with payload:
        """
        {}
        """

        Then I should have a gRPC response with code "Internal"
