Feature: List Items with internal server error

    Scenario: Internal Server Error
        When I request a gRPC method "/grpctest.ItemService/ListItems" with payload:
        """
        {}
        """

        Then I should have a gRPC response with code "INTERNAL"
        And I should have a gRPC response with error message "Internal Server Error"
