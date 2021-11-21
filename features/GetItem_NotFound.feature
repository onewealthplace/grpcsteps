Feature: Get Item

    Scenario: Item not found
        When I request a gRPC method "/grpctest.ItemService/GetItem" with payload:
        """
        {
            "id": 42
        }
        """

        Then I should have a gRPC response with code "NOT_FOUND"
        Then I should have a gRPC response with error message "Item 42 not found"
