Feature: List Items

    Scenario: Internal Server Error
        When I request a GRPC method "/grpctest.ItemService/ListItems" with payload:
        """
        {}
        """

        Then I should have a GRPC response with code "INTERNAL"
        Then I should have a GRPC response with error message "Internal Server Error"
