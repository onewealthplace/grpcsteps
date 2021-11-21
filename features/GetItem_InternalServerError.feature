Feature: Get Item

    Scenario: Internal Server Error
        When I request a gRPC method "/grpctest.ItemService/GetItem" with payload:
        """
        {
            "id": 42
        }
        """

        Then I should have a gRPC response with code "INTERNAL"
        Then I should have a gRPC response with error message "Internal Server Error"
